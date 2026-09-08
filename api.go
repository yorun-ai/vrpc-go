package vrpc

import (
	"time"

	internalvrpc "go.yorun.ai/vrpc/internal"
)

// Option configures a standalone client and its transport and credentials.
type Option = internalvrpc.Option

// Client invokes vRPC methods without a Vine application runtime.
type Client = internalvrpc.Client

// InvokeOption configures one invocation.
type InvokeOption = internalvrpc.InvokeOption

// ResponseMetadata contains HTTP and protocol metadata for an invocation.
type ResponseMetadata = internalvrpc.ResponseMetadata

// Identity describes a client or server instance.
type Identity = internalvrpc.Identity

// Trace contains a vRPC trace ID and span ID.
type Trace = internalvrpc.Trace

// ErrorPayload contains the structured error returned by the remote service.
type ErrorPayload = internalvrpc.ErrorPayload

// InvocationError preserves a failing remote status and its optional error payload.
type InvocationError = internalvrpc.InvocationError

// ProtocolError reports a malformed successful response.
type ProtocolError = internalvrpc.ProtocolError

// MethodSpec describes the wire name and binary flags of a client method.
type MethodSpec = internalvrpc.MethodSpec

// ServiceSpec describes a client service contract, without server registration.
type ServiceSpec = internalvrpc.ServiceSpec

// MethodInfo is an immutable snapshot of a registered client method.
type MethodInfo = internalvrpc.MethodInfo

// Registry stores client contracts and supports concurrent registration and lookup.
// Its zero value is ready to use.
type Registry = internalvrpc.Registry

// NewClient validates options and constructs a client. Invocations preserve
// context cancellation and do not add retries.
func NewClient(option Option) (*Client, error) {
	return internalvrpc.NewClient(option)
}

// WithTimeout sets the local and advertised timeout for one invocation.
func WithTimeout(timeout time.Duration) InvokeOption {
	return internalvrpc.WithTimeout(timeout)
}

// WithTrace creates a child span of parent for one invocation.
func WithTrace(parent Trace) InvokeOption {
	return internalvrpc.WithTrace(parent)
}

// NewRegistry creates an isolated client registry.
func NewRegistry() *Registry {
	return internalvrpc.NewRegistry()
}

// Register snapshots a client service in the default registry.
// Invalid or duplicate contracts panic.
func Register(spec *ServiceSpec) {
	internalvrpc.Register(spec)
}

// GetMethodInfo looks up a client method in the default registry.
func GetMethodInfo(serviceSkelName, methodSkelName string) (MethodInfo, bool) {
	return internalvrpc.GetMethodInfo(serviceSkelName, methodSkelName)
}

// EncodeIdentity validates and encodes a client or server instance identity.
func EncodeIdentity(identity Identity) (string, error) {
	return internalvrpc.EncodeIdentity(identity)
}

// DecodeIdentity validates and decodes a client or server identity header.
func DecodeIdentity(value string) (Identity, error) {
	return internalvrpc.DecodeIdentity(value)
}

// NewTrace creates a new trace and span.
func NewTrace() Trace {
	return internalvrpc.NewTrace()
}

// EncodeTrace validates and encodes a trace. A span is required.
func EncodeTrace(trace Trace) (string, error) {
	return internalvrpc.EncodeTrace(trace)
}

// EncodeCredentials formats Portal actor credentials as "field value, field value".
// A nil or empty map produces an empty value for anonymous calls. Ambiguous or
// unrepresentable fields fail; authentication remains the gateway's responsibility.
func EncodeCredentials(credentials map[string]string) (string, error) {
	return internalvrpc.EncodeCredentials(credentials)
}
