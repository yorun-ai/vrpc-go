package http

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
)

// ResponsePayload holds raw result/error bytes and their payload decoder.
// Schema-aware adapters retain ownership of typed decoding and validation.
type ResponsePayload struct {
	ResultBytes []byte
	ErrorBytes  []byte
	Unmarshal   func([]byte, any) error
}

// JSONRequest is the raw JSON request envelope extracted from Vine.
type JSONRequest struct {
	Params jsontext.Value `json:"params"`
}

// JSONResponse is the raw JSON response envelope extracted from Vine.
type JSONResponse struct {
	Result jsontext.Value `json:"result"`
	Error  jsontext.Value `json:"error"`
}

// EncodeJSONRequest wraps already-encoded arguments without changing schema encoding.
func EncodeJSONRequest(params []byte) ([]byte, error) {
	return json.Marshal(&JSONRequest{Params: params})
}

// DecodeJSONRequest extracts encoded arguments using Vine's request diagnostics.
func DecodeJSONRequest(body []byte) ([]byte, error) {
	var payload JSONRequest
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("request body cannot be parsed")
	}
	if len(payload.Params) == 0 {
		return nil, fmt.Errorf("missing request params")
	}
	return payload.Params, nil
}

// EncodeJSONResponse wraps encoded result/error values without re-encoding them.
func EncodeJSONResponse(result, err []byte) ([]byte, error) {
	return json.Marshal(&JSONResponse{Result: result, Error: err})
}

// DecodeJSONResponse extracts raw response payloads using Vine's wire representation.
func DecodeJSONResponse(body []byte) (*ResponsePayload, error) {
	var payload JSONResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("response body cannot be parsed")
	}
	return &ResponsePayload{ResultBytes: payload.Result, ErrorBytes: payload.Error, Unmarshal: func(data []byte, target any) error { return json.Unmarshal(data, target) }}, nil
}

// IsEmptyErrorPayload recognizes absent, JSON null, and CBOR null error payloads.
func IsEmptyErrorPayload(value []byte) bool {
	return len(value) == 0 || string(value) == "null" || len(value) == 1 && value[0] == 0xf6
}
