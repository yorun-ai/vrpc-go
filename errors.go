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
	Response *Response
	Payload  *ErrorPayload
	Cause    error
}

// Error returns the remote outcome and human-readable message.
func (e *InvocationError) Error() string {
	message := "remote invocation failed"
	if e.Payload != nil && e.Payload.Message != "" {
		message = e.Payload.Message
	}
	return fmt.Sprintf("vrpc: %s (status=%s, HTTP=%d)", message, e.Response.Status, e.Response.HTTPStatus)
}

// Unwrap returns a response decoding error, if any.
func (e *InvocationError) Unwrap() error { return e.Cause }

// ProtocolError indicates a missing or malformed protocol response.
// Response preserves HTTP status, headers, and any decoded metadata.
type ProtocolError struct {
	Response *Response
	Cause    error
}

// Error describes the malformed response without including its raw body.
func (e *ProtocolError) Error() string {
	return fmt.Sprintf("vrpc: invalid response (HTTP=%d): %v", e.Response.HTTPStatus, e.Cause)
}

// Unwrap returns the underlying protocol error.
func (e *ProtocolError) Unwrap() error { return e.Cause }
