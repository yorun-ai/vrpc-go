package http

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"net/http"

	"github.com/fxamacker/cbor/v2"
)

// ResponsePayload holds raw result/error bytes and their payload decoder.
// Schema-aware adapters retain ownership of typed decoding and validation.
type ResponsePayload struct {
	ResultBytes []byte
	ErrorBytes  []byte
	Unmarshal   func([]byte, any) error
}

// JSONResponse is the raw vRPC JSON response envelope.
type JSONResponse struct {
	Result jsontext.Value `json:"result"`
	Error  jsontext.Value `json:"error"`
}

// EncodeJSONResponse wraps encoded result/error values without re-encoding them.
func EncodeJSONResponse(result, err []byte) ([]byte, error) {
	return json.Marshal(&JSONResponse{
		Result: result,
		Error:  err,
	})
}

// DecodeJSONResponse extracts raw payloads from a JSON response envelope.
func DecodeJSONResponse(body []byte) (*ResponsePayload, error) {
	var payload JSONResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("response body cannot be parsed")
	}
	return &ResponsePayload{
		ResultBytes: payload.Result,
		ErrorBytes:  payload.Error,
		Unmarshal:   func(data []byte, target any) error { return json.Unmarshal(data, target) },
	}, nil
}

// IsEmptyErrorPayload recognizes absent, JSON null, and CBOR null error payloads.
func IsEmptyErrorPayload(value []byte) bool {
	return len(value) == 0 || string(value) == "null" || len(value) == 1 && value[0] == 0xf6
}

// CBORResponse is the raw vRPC CBOR response envelope.
type CBORResponse struct {
	Result cbor.RawMessage `json:"result"`
	Error  cbor.RawMessage `json:"error"`
}

// EncodeCBORResponse wraps schema-encoded CBOR result/error values.
func EncodeCBORResponse(result, err []byte) ([]byte, error) {
	return cbor.Marshal(&CBORResponse{
		Result: result,
		Error:  err,
	})
}

// DecodeCBORResponse extracts a response using the default CBOR decoder.
func DecodeCBORResponse(body []byte) (*ResponsePayload, error) {
	return DecodeCBORResponseWith(body, cbor.Unmarshal)
}

// DecodeCBORResponseWith allows a caller to use a stricter CBOR decoding profile.
func DecodeCBORResponseWith(body []byte, unmarshal func([]byte, any) error) (*ResponsePayload, error) {
	var payload CBORResponse
	if err := unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("response body cannot be parsed: %w", err)
	}
	return &ResponsePayload{
		ResultBytes: payload.Result,
		ErrorBytes:  payload.Error,
		Unmarshal:   unmarshal,
	}, nil
}

// ReadResponseBody reads the Rpc response body up to the fixed transport limit.
func ReadResponseBody(response *http.Response) ([]byte, error) {
	return ReadBody(response.Body, response.ContentLength, MaxResponseBodyBytes, "response")
}
