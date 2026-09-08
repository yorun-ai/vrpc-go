package http

import (
	"fmt"
	"net/http"
)

// DecodeError distinguishes response decoding failures from network failures.
// Framework adapters can translate these into their existing error model.
type DecodeError struct {
	Cause error
}

// Error describes the response decoding failure.
func (e *DecodeError) Error() string {
	return fmt.Sprintf("decode vRPC response: %v", e.Cause)
}

// Unwrap returns the decoder's original error.
func (e *DecodeError) Unwrap() error {
	return e.Cause
}

// RoundTrip runs one prepared HTTP request and owns the response body lifetime.
// It neither retries nor selects a network
// transport: standalone clients and h2c/mTLS adapters supply their own do function.
func RoundTrip[T any](request *http.Request, prepared func(), do func(*http.Request) (*http.Response, error), decode func(*http.Response) (T, error)) (T, error) {
	var zero T
	if prepared != nil {
		prepared()
	}
	response, err := do(request)
	if err != nil {
		return zero, err
	}
	defer response.Body.Close()
	result, err := decode(response)
	if err != nil {
		return result, &DecodeError{
			Cause: err,
		}
	}
	return result, nil
}
