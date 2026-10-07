package http

import (
	"encoding/json/jsontext"
	"fmt"
	"net/http"

	"github.com/fxamacker/cbor/v2"
)

// ErrorPayload is the structured error carried by a vRPC response.
// The response status determines the invocation outcome; Code is auxiliary data.
type ErrorPayload struct {
	Code    string `json:"code" cbor:"code"`
	Message string `json:"message" cbor:"message"`
	Reason  string `json:"reason" cbor:"reason"`
	Detail  string `json:"detail" cbor:"detail"`
}

// ResponsePayload holds raw result/error bytes and their payload decoder.
// Framework adapters retain ownership of typed decoding and validation.
type ResponsePayload struct {
	ResultBytes []byte
	ErrorBytes  []byte
	codec       _Codec
}

// Unmarshal decodes a result or error using the response's fixed wire rules.
// The target may be partially populated on error and must then be discarded.
func (p *ResponsePayload) Unmarshal(data []byte, target any) error {
	return p.codec.unmarshal(data, target)
}

// DecodeError decodes the shared error payload, returning nil for an absent or null error.
// It leaves status interpretation and application-specific code validation to adapters.
func (p *ResponsePayload) DecodeError() (*ErrorPayload, error) {
	if IsEmptyErrorPayload(p.ErrorBytes) {
		return nil, nil
	}
	var payload ErrorPayload
	if err := p.Unmarshal(p.ErrorBytes, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// EncodeWithError replaces the error value while preserving the encoded result.
func (p *ResponsePayload) EncodeWithError(errorValue *ErrorPayload) ([]byte, error) {
	raw, err := p.codec.marshal(errorValue)
	if err != nil {
		return nil, err
	}
	if p.codec {
		return EncodeCBORResponse(p.ResultBytes, raw)
	}
	return EncodeJSONResponse(p.ResultBytes, raw)
}

// EncodeResponse encodes the result, error and envelope with the vRPC wire rules.
// Adapters decide whether a response is successful and supply nil for no error.
func EncodeResponse(result any, errorValue *ErrorPayload, contentType string) ([]byte, error) {
	codec, err := codecForContentType(contentType)
	if err != nil {
		return nil, err
	}

	raw, err := codec.marshal(result)
	if err != nil {
		return nil, err
	}
	payload := ResponsePayload{
		ResultBytes: raw,
		codec:       codec,
	}
	return payload.EncodeWithError(errorValue)
}

// DecodeResponse extracts raw result/error values and their fixed wire decoder.
// Adapters decode the required values with ResponsePayload.Unmarshal.
func DecodeResponse(body []byte, contentType string) (*ResponsePayload, error) {
	codec, err := codecForContentType(contentType)
	if err != nil {
		return nil, err
	}
	if codec {
		return DecodeCBORResponse(body)
	}
	return DecodeJSONResponse(body)
}

// JSONResponse is the raw vRPC JSON response envelope.
type JSONResponse struct {
	Result jsontext.Value `json:"result"`
	Error  jsontext.Value `json:"error"`
}

// EncodeJSONResponse wraps encoded result/error values without re-encoding them.
func EncodeJSONResponse(result, err []byte) ([]byte, error) {
	return _Codec(false).marshal(&JSONResponse{
		Result: result,
		Error:  err,
	})
}

// DecodeJSONResponse extracts raw payloads from a JSON response envelope.
func DecodeJSONResponse(body []byte) (*ResponsePayload, error) {
	var payload JSONResponse
	if err := _Codec(false).unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("response body cannot be parsed")
	}
	return &ResponsePayload{
		ResultBytes: payload.Result,
		ErrorBytes:  payload.Error,
		codec:       false,
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

// EncodeCBORResponse wraps already-encoded CBOR result/error values.
func EncodeCBORResponse(result, err []byte) ([]byte, error) {
	return _Codec(true).marshal(&CBORResponse{
		Result: result,
		Error:  err,
	})
}

// DecodeCBORResponse rejects duplicate envelope keys and provides a decoder
// that also rejects duplicate keys in the result and error payloads.
func DecodeCBORResponse(body []byte) (*ResponsePayload, error) {
	var payload CBORResponse
	if err := _Codec(true).unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("response body cannot be parsed: %w", err)
	}
	return &ResponsePayload{
		ResultBytes: payload.Result,
		ErrorBytes:  payload.Error,
		codec:       true,
	}, nil
}

// ReadResponseBody reads the Rpc response body up to the fixed transport limit.
func ReadResponseBody(response *http.Response) ([]byte, error) {
	return ReadBody(response.Body, response.ContentLength, MaxResponseBodyBytes, "response")
}
