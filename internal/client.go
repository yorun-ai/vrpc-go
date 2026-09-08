package vrpc

import (
	"context"
	"errors"
	"fmt"
	rpchttp "go.yorun.ai/vrpc/transport/http"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Option configures a standalone client. Endpoint includes the invocation prefix.
type Option struct {
	Endpoint string
	// Identity requires a name, version and instance UUID.
	Identity Identity
	// Transport defaults to http.DefaultTransport. The caller owns its lifecycle.
	Transport http.RoundTripper
	// Timeout defaults to 30 seconds. A shorter context deadline takes precedence.
	Timeout time.Duration
	// Headers are copied at construction. Client-generated headers take precedence.
	Headers http.Header
	// Authorization is encoded once; a non-nil map overrides the Authorization header.
	Authorization map[string]string
}

type ResponseMetadata struct {
	HTTPStatus int
	Status     string
	Header     http.Header
	Server     Identity
}

type Client struct {
	endpoint string
	identity string
	http     *http.Client
	timeout  time.Duration
	headers  http.Header
}

func NewClient(option Option) (*Client, error) {
	endpoint, err := url.Parse(option.Endpoint)
	if err != nil || endpoint == nil || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.Host == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.ForceQuery || endpoint.Fragment != "" {
		return nil, fmt.Errorf("vrpc: endpoint must be an absolute HTTP(S) URL without credentials, query, or fragment")
	}

	encodedIdentity, err := EncodeIdentity(option.Identity)
	if err != nil {
		return nil, err
	}

	if option.Timeout < 0 {
		return nil, fmt.Errorf("vrpc: timeout cannot be negative")
	}
	if option.Timeout == 0 {
		option.Timeout = 30 * time.Second
	}

	headers := make(http.Header, len(option.Headers))
	for name, values := range option.Headers {
		key := http.CanonicalHeaderKey(name)
		headers[key] = append(headers[key], values...)
	}

	if option.Authorization != nil {
		authorization, err := EncodeCredentials(option.Authorization)
		if err != nil {
			return nil, err
		}
		if authorization == "" {
			headers.Del("Authorization")
		} else {
			headers.Set("Authorization", authorization)
		}
	}

	return &Client{
		endpoint: strings.TrimRight(endpoint.String(), "/"),
		identity: encodedIdentity,
		http: &http.Client{
			Transport: option.Transport,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		timeout: option.Timeout,
		headers: headers,
	}, nil
}

type InvokeOption func(*_InvokeOptions) error
type _InvokeOptions struct {
	timeout  time.Duration
	trace    Trace
	hasTrace bool
}

func WithTimeout(timeout time.Duration) InvokeOption {
	return func(o *_InvokeOptions) error {
		if timeout <= 0 {
			return fmt.Errorf("vrpc: timeout must be positive")
		}
		o.timeout = timeout
		return nil
	}
}

func WithTrace(parent Trace) InvokeOption {
	return func(o *_InvokeOptions) error {
		if _, err := EncodeTrace(parent); err != nil {
			return err
		}
		o.trace = parent.Child()
		o.hasTrace = true
		return nil
	}
}

// Invoke returns the typed result and response metadata, or a zero result on failure.
func (client *Client) Invoke[T any](ctx context.Context, method MethodInfo, params any, options ...InvokeOption) (T, *ResponseMetadata, error) {
	var result T
	response, err := client.invoke(ctx, method, params, &result, options...)
	if err != nil {
		var zero T
		return zero, response, err
	}
	return result, response, nil
}

// InvokeRaw calls an unregistered service/method using JSON and returns a dynamic result.
func (client *Client) InvokeRaw(ctx context.Context, service, method string, params any, options ...InvokeOption) (any, *ResponseMetadata, error) {
	path := "/" + service + "/" + method
	if _, _, err := rpchttp.ParseServiceAndMethodFromPath(path); err != nil {
		return nil, nil, err
	}

	info := MethodInfo{
		service: service,
		path:    path,
		spec: MethodSpec{
			SkelName: method,
		},
	}
	return client.Invoke[any](ctx, info, params, options...)
}

func (c *Client) invoke(ctx context.Context, method MethodInfo, params, result any, options ...InvokeOption) (*ResponseMetadata, error) {
	if method.path == "" {
		return nil, fmt.Errorf("vrpc: invalid method info")
	}

	invocation := _InvokeOptions{
		timeout: c.timeout,
	}
	for _, option := range options {
		if err := option(&invocation); err != nil {
			return nil, err
		}
	}
	if !invocation.hasTrace {
		invocation.trace = NewTrace()
	}

	ctx, cancel := context.WithTimeout(ctx, invocation.timeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	encoding := method.invokeEncoding()
	body, err := encoding.request.EncodeRequest(params)
	if err != nil {
		return nil, fmt.Errorf("vrpc: encode request: %w", err)
	}

	request, err := rpchttp.NewRequest(ctx, c.endpoint, method.FullURLPath(), body, encoding.request.ContentType(), encoding.accept)
	if err != nil {
		return nil, err
	}

	headers := c.headers.Clone()
	for name, values := range request.Header {
		headers[name] = values
	}
	request.Header = headers
	request.Header.Set(rpchttp.HeaderRpcClient, c.identity)
	trace, _ := EncodeTrace(invocation.trace)
	request.Header.Set(rpchttp.HeaderRpcTrace, trace)

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	rpchttp.EncodeRequestOptionsToHeader(request.Header, ctx)

	metadata, err := rpchttp.RoundTrip(request, nil, c.http.Do, func(response *http.Response) (*ResponseMetadata, error) {
		metadata := &ResponseMetadata{
			HTTPStatus: response.StatusCode,
			Status:     response.Header.Get(rpchttp.HeaderRpcStatus),
			Header:     response.Header.Clone(),
		}

		payload, decodeErr := c.decodeResponse(response, metadata, result, encoding)
		if (metadata.Status != "" && metadata.Status != rpchttp.StatusOK) || response.StatusCode < 200 || response.StatusCode >= 300 {
			return metadata, &InvocationError{
				Metadata: metadata,
				Payload:  payload,
				Cause:    decodeErr,
			}
		}

		if decodeErr != nil {
			return metadata, &ProtocolError{
				Metadata: metadata,
				Cause:    decodeErr,
			}
		}

		if payload != nil {
			return metadata, &ProtocolError{
				Metadata: metadata,
				Cause:    fmt.Errorf("error payload conflicts with OK status"),
			}
		}

		return metadata, nil
	})
	if decodeErr, ok := errors.AsType[*rpchttp.DecodeError](err); ok {
		return metadata, decodeErr.Cause
	}
	if err != nil {
		return nil, fmt.Errorf("vrpc: transport: %w", err)
	}

	return metadata, nil
}

func (c *Client) decodeResponse(response *http.Response, metadata *ResponseMetadata, result any, encoding _InvokeEncoding) (*ErrorPayload, error) {
	if err := rpchttp.CheckResponseHeaders(response.Header); err != nil {
		return nil, err
	}
	if !statusPattern.MatchString(metadata.Status) {
		return nil, fmt.Errorf("invalid status")
	}

	server, err := DecodeIdentity(response.Header.Get(rpchttp.HeaderRpcServer))
	if err != nil {
		return nil, err
	}
	metadata.Server = server

	contentType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}
	codec := _Codec(false)
	if contentType != rpchttp.ContentTypeJson {
		if !encoding.binary || contentType != rpchttp.ContentTypeCbor {
			return nil, fmt.Errorf("unsupported response content type %s", contentType)
		}
		codec = _Codec(true)
	}

	if encoding := response.Header.Get("Content-Encoding"); encoding != "" && encoding != "identity" {
		return nil, fmt.Errorf("unsupported response content encoding %s", encoding)
	}

	body, err := rpchttp.ReadResponseBody(response)
	if err != nil {
		return nil, err
	}

	if metadata.Status != rpchttp.StatusOK || metadata.HTTPStatus < 200 || metadata.HTTPStatus >= 300 {
		result = nil
	}

	return codec.DecodeResponse(body, result)
}
