package vrpc

import "fmt"

// ErrorPayload is the auxiliary error object returned by a vRPC service.
// Status in the response header, rather than Code here, identifies the outcome.
type ErrorPayload struct {
	Type    string `json:"type" cbor:"type"`
	Code    string `json:"code" cbor:"code"`
	Reason  string `json:"reason" cbor:"reason"`
	Message string `json:"message" cbor:"message"`
	Detail  string `json:"detail" cbor:"detail"`
}

// InvocationError represents a failing vRPC status or a non-success HTTP status.
// Payload may be nil. Cause preserves any response decoding failure.
type InvocationError struct {
	Metadata *ResponseMetadata
	Payload  *ErrorPayload
	Cause    error
}

func (e *InvocationError) Error() string {
	message := "remote invocation failed"
	if e.Payload != nil && e.Payload.Message != "" {
		message = e.Payload.Message
	}
	return fmt.Sprintf("vrpc: %s (status=%s, HTTP=%d)", message, e.Metadata.Status, e.Metadata.HTTPStatus)
}

func (e *InvocationError) Unwrap() error {
	return e.Cause
}

// ProtocolError indicates a missing or malformed protocol response.
// Metadata preserves HTTP status, headers, and any decoded metadata.
type ProtocolError struct {
	Metadata *ResponseMetadata
	Cause    error
}

func (e *ProtocolError) Error() string {
	return fmt.Sprintf("vrpc: invalid response (HTTP=%d): %v", e.Metadata.HTTPStatus, e.Cause)
}

func (e *ProtocolError) Unwrap() error {
	return e.Cause
}
