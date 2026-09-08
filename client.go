package vrpc

import (
	"context"
	"errors"
	"fmt"
	wire "go.yorun.ai/vrpc/transport/http"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"
	"uuid"
)

// Options configures a standalone client. Endpoint includes the invocation prefix.
type Options struct {
	Endpoint string
	// Identity defaults to vrpc.client, version 0.0.0, and a random instance UUID.
	Identity Identity
	// Transport defaults to http.DefaultTransport. The caller owns its lifecycle.
	Transport http.RoundTripper
	// Timeout defaults to 30 seconds. A shorter context deadline takes precedence.
	Timeout time.Duration
	// Headers are copied at construction. Protocol and transport headers are reserved.
	Headers http.Header
	// Authorization supplies the complete Authorization value for each call.
	// The callback may refresh credentials and must respect its context.
	Authorization func(context.Context) (string, error)
	// Codec defaults to JSONCodec. A non-JSON codec also accepts JSON responses.
	Codec Codec
	// MaxResponseBytes limits the decoded HTTP body; zero defaults to 16 MiB.
	MaxResponseBytes int64
}

// Response contains metadata available even when a remote invocation fails.
type Response struct {
	HTTPStatus    int
	Status        string
	Header        http.Header
	Server        Identity
	Trace         Trace
	PortalTraceID string
}

// Client invokes vRPC methods without a Vine runtime. Use NewClient to construct it.
type Client struct {
	endpoint         string
	identity         string
	http             *http.Client
	timeout          time.Duration
	headers          http.Header
	authorization    func(context.Context) (string, error)
	codec            Codec
	maxResponseBytes int64
}

// NewClient validates options and creates a client. Redirects are not followed:
// a redirected POST may change semantics or disclose credentials.
func NewClient(options Options) (*Client, error) {
	endpoint, err := url.Parse(options.Endpoint)
	if err != nil || endpoint == nil || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.Host == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.ForceQuery || endpoint.Fragment != "" {
		return nil, fmt.Errorf("vrpc: endpoint must be an absolute HTTP(S) URL without credentials, query, or fragment")
	}
	identity := options.Identity
	if identity.Name == "" {
		identity.Name = "vrpc.client"
	}
	if identity.Version == "" {
		identity.Version = "0.0.0"
	}
	if identity.InstanceID == "" {
		identity.InstanceID = uuid.NewV7().String()
	}
	encodedIdentity, err := EncodeIdentity(identity)
	if err != nil {
		return nil, err
	}
	if options.Timeout < 0 || options.MaxResponseBytes < 0 {
		return nil, fmt.Errorf("vrpc: timeout and response limit cannot be negative")
	}
	if options.Timeout == 0 {
		options.Timeout = 30 * time.Second
	}
	if options.MaxResponseBytes == 0 {
		options.MaxResponseBytes = 16 << 20
	}
	if options.MaxResponseBytes == int64(^uint64(0)>>1) {
		return nil, fmt.Errorf("vrpc: response limit is too large")
	}
	if err := validateHeaders(options.Headers); err != nil {
		return nil, err
	}
	headers := make(http.Header, len(options.Headers))
	for name, values := range options.Headers {
		headers.Set(name, values[0])
	}
	codec := options.Codec
	if codec == nil {
		codec = JSONCodec{}
	}
	if codec.ContentType() != ContentTypeJSON && codec.ContentType() != ContentTypeCBOR {
		return nil, fmt.Errorf("vrpc: unsupported codec content type")
	}
	return &Client{endpoint: strings.TrimRight(endpoint.String(), "/"), identity: encodedIdentity,
		http:    &http.Client{Transport: options.Transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		timeout: options.Timeout, headers: headers, authorization: options.Authorization, codec: codec, maxResponseBytes: options.MaxResponseBytes}, nil
}

// CallOption configures one invocation.
type CallOption func(*_CallOptions) error
type _CallOptions struct {
	timeout  time.Duration
	trace    Trace
	hasTrace bool
}

// WithTimeout sets the local and advertised timeout for one call.
func WithTimeout(timeout time.Duration) CallOption {
	return func(o *_CallOptions) error {
		if timeout <= 0 {
			return fmt.Errorf("vrpc: timeout must be positive")
		}
		o.timeout = timeout
		return nil
	}
}

// WithTrace creates a child span of parent for this call, preserving its trace ID.
func WithTrace(parent Trace) CallOption {
	return func(o *_CallOptions) error {
		if _, err := EncodeTrace(parent); err != nil {
			return err
		}
		o.trace = parent.Child()
		o.hasTrace = true
		return nil
	}
}

// Call invokes service/method with params and decodes its result into a pointer.
// Pass nil result for a void method. Metadata is returned for every HTTP response.
// Transport and context failures wrap their causes for errors.Is/errors.As.
// Client cancellation does not guarantee cancellation of execution behind Portal.
func (c *Client) Call(ctx context.Context, service, method string, params, result any, options ...CallOption) (*Response, error) {
	if _, _, err := wire.ParseServiceAndMethodFromPath("/" + service + "/" + method); err != nil || service == "." || service == ".." {
		return nil, fmt.Errorf("vrpc: invalid service or method name")
	}
	call := _CallOptions{timeout: c.timeout}
	for _, option := range options {
		if err := option(&call); err != nil {
			return nil, err
		}
	}
	if !call.hasTrace {
		call.trace = NewTrace()
	}
	ctx, cancel := context.WithTimeout(ctx, call.timeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	body, err := c.codec.EncodeRequest(params)
	if err != nil {
		return nil, fmt.Errorf("vrpc: encode request: %w", err)
	}
	accept := ContentTypeJSON
	if c.codec.ContentType() != ContentTypeJSON {
		accept = c.codec.ContentType() + ", " + ContentTypeJSON
	}
	request, err := wire.NewRequest(ctx, c.endpoint, "/"+service+"/"+method, body, c.codec.ContentType(), accept)
	if err != nil {
		return nil, err
	}
	for name, values := range c.headers {
		request.Header[name] = append([]string(nil), values...)
	}
	request.Header.Set(HeaderClient, c.identity)
	trace, _ := EncodeTrace(call.trace)
	request.Header.Set(HeaderTrace, trace)
	if c.authorization != nil {
		value, err := c.authorization(ctx)
		if err != nil {
			return nil, fmt.Errorf("vrpc: authorization: %w", err)
		}
		if value == "" {
			request.Header.Del("Authorization")
		} else {
			request.Header.Set("Authorization", value)
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	wire.EncodeRequestOptionsToHeader(request.Header, ctx)
	metadata, err := wire.Exchange(request, nil, c.http.Do, func(response *http.Response) (*Response, error) {
		metadata := &Response{HTTPStatus: response.StatusCode, Status: response.Header.Get(HeaderStatus), Header: response.Header.Clone(), Trace: call.trace, PortalTraceID: response.Header.Get(HeaderPortalTraceID)}
		payload, decodeErr := c.decodeResponse(response, metadata, result)
		if (metadata.Status != "" && metadata.Status != StatusOK) || response.StatusCode < 200 || response.StatusCode >= 300 {
			return metadata, &InvocationError{Response: metadata, Payload: payload, Cause: decodeErr}
		}
		if decodeErr != nil {
			return metadata, &ProtocolError{Response: metadata, Cause: decodeErr}
		}
		if payload != nil {
			return metadata, &ProtocolError{Response: metadata, Cause: fmt.Errorf("error payload conflicts with OK status")}
		}
		return metadata, nil
	})
	if decodeErr, ok := errors.AsType[*wire.DecodeError](err); ok {
		return metadata, decodeErr.Cause
	}
	if err != nil {
		return nil, fmt.Errorf("vrpc: transport: %w", err)
	}
	return metadata, nil
}

func (c *Client) decodeResponse(response *http.Response, metadata *Response, result any) (*ErrorPayload, error) {
	if err := wire.CheckResponseHeaders(response.Header); err != nil {
		return nil, err
	}
	if !statusPattern.MatchString(metadata.Status) {
		return nil, fmt.Errorf("invalid status")
	}
	server, err := DecodeIdentity(response.Header.Get(HeaderServer))
	if err != nil {
		return nil, err
	}
	metadata.Server = server
	contentType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}
	var codec Codec = JSONCodec{}
	if contentType != ContentTypeJSON {
		if contentType != c.codec.ContentType() {
			return nil, fmt.Errorf("unsupported response content type %s", contentType)
		}
		codec = c.codec
	}
	if encoding := response.Header.Get("Content-Encoding"); encoding != "" && encoding != "identity" {
		return nil, fmt.Errorf("unsupported response content encoding %s", encoding)
	}
	body, err := wire.ReadBody(response.Body, response.ContentLength, c.maxResponseBytes, "response")
	if err != nil {
		return nil, err
	}
	if metadata.Status != StatusOK || metadata.HTTPStatus < 200 || metadata.HTTPStatus >= 300 {
		result = nil
	}
	return codec.DecodeResponse(body, result)
}

func validateHeaders(headers http.Header) error {
	seen := map[string]bool{}
	for name, values := range headers {
		key := strings.ToLower(name)
		if seen[key] {
			return fmt.Errorf("vrpc: duplicated header %s", name)
		}
		seen[key] = true
		switch key {
		case "accept", "content-type", "content-length", "content-encoding", "accept-encoding", "host", "connection", "transfer-encoding", "trailer", "upgrade", "idempotency-key", "x-idempotency-key":
			return fmt.Errorf("vrpc: reserved header %s", name)
		}
		if strings.HasPrefix(key, "vrpc-") {
			return fmt.Errorf("vrpc: reserved header %s", name)
		}
		if len(values) != 1 {
			return fmt.Errorf("vrpc: header %s must have exactly one value", name)
		}
	}
	return nil
}
