// Package cbor supplies the optional CBOR wire envelopes extracted from Vine's HTTP transport.
package cbor

import (
	"fmt"
	"github.com/fxamacker/cbor/v2"
	wire "go.yorun.ai/vrpc/transport/http"
)

// Request is the raw CBOR request envelope extracted from Vine.
type Request struct {
	Params cbor.RawMessage `json:"params"`
}

// Response is the raw CBOR response envelope extracted from Vine.
type Response struct {
	Result cbor.RawMessage `json:"result"`
	Error  cbor.RawMessage `json:"error"`
}

// EncodeRequest wraps schema-encoded CBOR parameters without modifying their representation.
func EncodeRequest(params []byte) ([]byte, error) { return cbor.Marshal(&Request{Params: params}) }

// DecodeRequest extracts raw CBOR parameters, preserving Vine request diagnostics.
func DecodeRequest(body []byte) ([]byte, error) {
	var payload Request
	if err := cbor.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("request body cannot be parsed")
	}
	if len(payload.Params) == 0 {
		return nil, fmt.Errorf("missing request params")
	}
	return payload.Params, nil
}

// EncodeResponse wraps schema-encoded CBOR result/error values.
func EncodeResponse(result, err []byte) ([]byte, error) {
	return cbor.Marshal(&Response{Result: result, Error: err})
}

// DecodeResponse extracts a response using Vine's CBOR decoder.
func DecodeResponse(body []byte) (*wire.ResponsePayload, error) {
	return DecodeResponseWith(body, cbor.Unmarshal)
}

// DecodeResponseWith allows a caller to use a stricter CBOR decoding profile.
func DecodeResponseWith(body []byte, unmarshal func([]byte, any) error) (*wire.ResponsePayload, error) {
	var payload Response
	if err := unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("response body cannot be parsed: %w", err)
	}
	return &wire.ResponsePayload{ResultBytes: payload.Result, ErrorBytes: payload.Error, Unmarshal: unmarshal}, nil
}
