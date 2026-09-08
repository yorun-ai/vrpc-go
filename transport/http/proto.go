// Package http contains the shared vRPC HTTP wire transport.
// Framework metadata, schema selection, actor authorization and error policies
// are supplied by adapters; this package has no Vine dependencies.
package http

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"
)

// HTTP method
const RequestMethod = http.MethodPost

// CheckRequestMethod validates the vRPC POST method.
func CheckRequestMethod(req *http.Request) error {
	if req.Method != RequestMethod {
		return fmt.Errorf("invalid request method, expected %s", RequestMethod)
	}
	return nil
}

// Header definitions
const (
	HeaderAccept         = "accept"
	HeaderAcceptEncoding = "accept-encoding"

	HeaderContentType     = "content-type"
	HeaderContentEncoding = "content-encoding"
	HeaderContentLength   = "content-length"

	HeaderRpcTrace = "vrpc-trace"

	HeaderRpcClient    = "vrpc-client"
	HeaderRpcActor     = "vrpc-actor"
	HeaderRpcInitiator = "vrpc-initiator"
	HeaderRpcOptions   = "vrpc-options"

	HeaderRpcStatus = "vrpc-status"
	HeaderRpcServer = "vrpc-server"
)

// Content types
const (
	ContentTypeJson = "application/vrpc+json"
	ContentTypeCbor = "application/vrpc+cbor"
)

var supportedContentTypes = []string{ContentTypeJson, ContentTypeCbor}

// MediaTypeOf extracts a media type, falling back to the value before a semicolon.
func MediaTypeOf(value string) string {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(value))
	if err == nil {
		return mediaType
	}
	mediaType, _, _ = strings.Cut(value, ";")
	return strings.TrimSpace(mediaType)
}

// IsValidContentType reports whether a media type is supported.
func IsValidContentType(value string) bool {
	value = MediaTypeOf(value)
	return slices.Contains(supportedContentTypes, value)
}

// AcceptsValidContentType reports whether Accept includes a supported media type.
func AcceptsValidContentType(value string) bool {
	for accepted := range strings.SplitSeq(value, ",") {
		if IsValidContentType(accepted) {
			return true
		}
	}
	return false
}

// AcceptsContentType reports whether Accept includes contentType.
func AcceptsContentType(value string, contentType string) bool {
	for accepted := range strings.SplitSeq(value, ",") {
		if MediaTypeOf(accepted) == contentType {
			return true
		}
	}
	return false
}

// Header checks and fixed headers
func headerValue(header http.Header, key string) (value string, ok bool) {
	if values := header.Values(key); len(values) > 0 {
		return values[0], true
	}
	return "", false
}

var (
	requiredRequestHeaders = []string{
		HeaderAccept,
		HeaderContentType,
		HeaderRpcTrace,
		HeaderRpcClient,
	}
	requiredResponseHeaders = []string{
		HeaderContentType,
		HeaderRpcStatus,
		HeaderRpcServer,
	}
)

func newMissingHeaderError(scope string, key string) error {
	return fmt.Errorf("missing %s header %s", scope, key)
}

func newDuplicatedHeaderError(scope string, key string) error {
	return fmt.Errorf("invalid %s header %s: duplicated", scope, key)
}

func newInvalidHeaderError(scope string, key string, value string) error {
	return fmt.Errorf("invalid %s header %s: %s", scope, key, value)
}

func checkDuplicatedHeaders(header http.Header, scope string) error {
	for key, values := range header {
		if len(values) > 1 {
			return newDuplicatedHeaderError(scope, key)
		}
	}
	return nil
}

func checkRequiredHeaders(header http.Header, scope string, required []string) error {
	for _, key := range required {
		_, ok := headerValue(header, key)
		if !ok {
			return newMissingHeaderError(scope, key)
		}
	}
	return nil
}

func checkContentTypeHeader(header http.Header, scope string) error {
	value, ok := headerValue(header, HeaderContentType)
	if !ok {
		return newMissingHeaderError(scope, HeaderContentType)
	}
	if !IsValidContentType(value) {
		return newInvalidHeaderError(scope, HeaderContentType, value)
	}
	return nil
}

// CheckRequestContentTypeHeader checks request content negotiation.
func CheckRequestContentTypeHeader(header http.Header) error {
	return checkContentTypeHeader(header, "request")
}

func checkAcceptHeader(header http.Header, scope string) error {
	value, ok := headerValue(header, HeaderAccept)
	if !ok {
		return newMissingHeaderError(scope, HeaderAccept)
	}
	if !AcceptsValidContentType(value) {
		return newInvalidHeaderError(scope, HeaderAccept, value)
	}
	return nil
}

// CheckRequestHeaders validates required request headers and duplicates.
func CheckRequestHeaders(header http.Header) error {
	if err := checkDuplicatedHeaders(header, "request"); err != nil {
		return err
	}
	if err := checkRequiredHeaders(header, "request", requiredRequestHeaders); err != nil {
		return err
	}
	if err := checkAcceptHeader(header, "request"); err != nil {
		return err
	}
	return checkContentTypeHeader(header, "request")
}

// CheckResponseHeaders validates required response headers and duplicates.
func CheckResponseHeaders(header http.Header) error {
	if err := checkDuplicatedHeaders(header, "response"); err != nil {
		return err
	}
	if err := checkRequiredHeaders(header, "response", requiredResponseHeaders); err != nil {
		return err
	}
	return checkContentTypeHeader(header, "response")
}

// Options describes the shared request timeout metadata.
type Options struct {
	Timeout time.Duration
}

// DecodeFields parses comma-delimited vRPC metadata.
func DecodeFields(value string) (map[string]string, error) {
	fields := map[string]string{}
	for part := range strings.SplitSeq(value, ",") {
		name, fieldValue, ok := strings.Cut(part, "=")
		if !ok {
			return nil, fmt.Errorf("missing value")
		}
		name = strings.TrimSpace(name)
		fieldValue = strings.TrimSpace(fieldValue)
		if name == "" || fieldValue == "" {
			return nil, fmt.Errorf("empty field")
		}
		if _, exists := fields[name]; exists {
			return nil, fmt.Errorf("duplicated field")
		}
		fields[name] = fieldValue
	}
	return fields, nil
}

// EncodeFields encodes validated metadata fields. Invalid pairs panic.
func EncodeFields(pairs ...string) string {
	if len(pairs)%2 != 0 {
		panic("delimited pairs must be even")
	}
	parts := make([]string, 0, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		name, value := pairs[i], pairs[i+1]
		if name == "" || value == "" || strings.ContainsAny(name, ",=") || strings.ContainsAny(value, ",=") {
			panic("invalid delimited field")
		}
		parts = append(parts, name+"="+value)
	}
	return strings.Join(parts, ",")
}

// EncodeApp formats a validated client/server identity without owning its model.
func EncodeApp(name, version, instanceID string) string {
	return EncodeFields("name", name, "version", version, "instanceId", instanceID)
}

// EncodeTrace formats validated trace identifiers without owning their lifecycle.
func EncodeTrace(id, span string) string {
	return EncodeFields("id", id, "span", span)
}

// DecodeOptionsFromHeader parses an optional positive timeout.
func DecodeOptionsFromHeader(header http.Header) (*Options, error) {
	value, ok := headerValue(header, HeaderRpcOptions)
	if !ok {
		return &Options{}, nil
	}

	fields, err := DecodeFields(value)
	if err != nil {
		return nil, fmt.Errorf("invalid request header %s", HeaderRpcOptions)
	}

	timeoutValue := fields["timeout"]
	if timeoutValue == "" || len(fields) != 1 {
		return nil, fmt.Errorf("invalid request header %s", HeaderRpcOptions)
	}

	timeout, err := time.ParseDuration(timeoutValue)
	if err != nil || timeout <= 0 {
		return nil, fmt.Errorf("invalid request header %s", HeaderRpcOptions)
	}

	return &Options{
		Timeout: timeout,
	}, nil
}

// EncodeOptionsToHeader encodes a positive timeout when configured.
func EncodeOptionsToHeader(header http.Header, options *Options) {
	if options == nil || options.Timeout <= 0 {
		return
	}
	header.Set(HeaderRpcOptions, EncodeFields("timeout", options.Timeout.String()))
}

// EncodeRequestOptionsToHeader forwards the remaining context deadline.
func EncodeRequestOptionsToHeader(header http.Header, ctx context.Context) {
	deadline, ok := ctx.Deadline()
	if !ok {
		return
	}
	timeout := time.Until(deadline)
	if timeout <= 0 {
		return
	}
	EncodeOptionsToHeader(header, &Options{
		Timeout: timeout,
	})
}

// Request path
var requestPathRegexp = regexp.MustCompile(`^/([\w.]+)/(\w+)$`)

// ParseServiceAndMethodFromPath parses only the RPC path segment. Host-level prefixes such
// as "/rpc/invoke" should be removed by the caller before the request reaches http transport.
func ParseServiceAndMethodFromPath(path string) (serviceName string, methodName string, err error) {
	ms := requestPathRegexp.FindStringSubmatch(path)
	if ms == nil {
		return "", "", fmt.Errorf("invalid request path: %s", path)
	}
	return ms[1], ms[2], nil
}

// RequestContentTypes selects Content-Type and Accept from generated method
// binary flags. Request and result encoding are selected separately.
func RequestContentTypes(argumentsContainBinary, resultContainsBinary bool) (contentType, accept string) {
	contentType, accept = ContentTypeJson, ContentTypeJson
	if argumentsContainBinary {
		contentType = ContentTypeCbor
	}
	if resultContainsBinary {
		accept = ContentTypeCbor + ", " + ContentTypeJson
	}
	return
}

const (
	// MaxRequestBodyBytes is the maximum encoded Rpc request body size.
	MaxRequestBodyBytes int64 = 32 << 20
	// MaxResponseBodyBytes is the maximum encoded Rpc response body size.
	MaxResponseBodyBytes int64 = 128 << 20
)

// ReadBody applies a byte limit to an HTTP body before and during reading.
func ReadBody(body io.Reader, contentLength int64, limit int64, scope string) ([]byte, error) {
	if contentLength > limit {
		return nil, fmt.Errorf("rpc %s body exceeds %d byte limit", scope, limit)
	}
	if body == nil {
		return []byte{}, nil
	}

	content, err := io.ReadAll(io.LimitReader(body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > limit {
		return nil, fmt.Errorf("rpc %s body exceeds %d byte limit", scope, limit)
	}
	return content, nil
}

// HeaderPortalTraceID carries the Portal trace identifier.
const HeaderPortalTraceID = "portal-trace-id"

// StatusOK identifies a successful vRPC response.
const StatusOK = "OK"
