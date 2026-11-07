package http

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"fmt"
	"net/http"

	"github.com/fxamacker/cbor/v2"
)

// EncodeRequest encodes arguments and their envelope with the vRPC wire rules.
func EncodeRequest(params any, contentType string) ([]byte, error) {
	codec, err := codecForContentType(contentType)
	if err != nil {
		return nil, err
	}

	raw, err := codec.marshal(params)
	if err != nil {
		return nil, err
	}
	if codec {
		return EncodeCBORRequest(raw)
	}
	return EncodeJSONRequest(raw)
}

// DecodeRequest decodes the envelope and arguments, rejecting duplicate keys.
// The target may be partially populated on error and must then be discarded.
func DecodeRequest(body []byte, params any, contentType string) error {
	codec, err := codecForContentType(contentType)
	if err != nil {
		return err
	}

	var raw []byte
	if codec {
		raw, err = DecodeCBORRequest(body)
	} else {
		raw, err = DecodeJSONRequest(body)
	}
	if err != nil {
		return err
	}

	if err := codec.unmarshal(raw, params); err != nil {
		return fmt.Errorf("request params cannot be parsed")
	}
	return nil
}

// JSONRequest is the raw vRPC JSON request envelope.
type JSONRequest struct {
	Params jsontext.Value `json:"params"`
}

// EncodeJSONRequest wraps already-encoded arguments without re-encoding them.
func EncodeJSONRequest(params []byte) ([]byte, error) {
	return _Codec(false).marshal(&JSONRequest{
		Params: params,
	})
}

// DecodeJSONRequest extracts encoded arguments from a JSON request envelope.
func DecodeJSONRequest(body []byte) ([]byte, error) {
	var payload JSONRequest
	if err := _Codec(false).unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("request body cannot be parsed")
	}
	if len(payload.Params) == 0 {
		return nil, fmt.Errorf("missing request params")
	}
	return payload.Params, nil
}

// CBORRequest is the raw vRPC CBOR request envelope.
type CBORRequest struct {
	Params cbor.RawMessage `json:"params"`
}

// EncodeCBORRequest wraps already-encoded CBOR parameters without modifying their representation.
func EncodeCBORRequest(params []byte) ([]byte, error) {
	return _Codec(true).marshal(&CBORRequest{
		Params: params,
	})
}

// DecodeCBORRequest extracts raw parameters, rejecting duplicate envelope keys.
// Callers must also reject duplicate keys when decoding the raw parameters.
func DecodeCBORRequest(body []byte) ([]byte, error) {
	var payload CBORRequest
	if err := _Codec(true).unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("request body cannot be parsed")
	}
	if len(payload.Params) == 0 {
		return nil, fmt.Errorf("missing request params")
	}
	return payload.Params, nil
}

// RequestBodyLimit returns the encoded request body limit for a Content-Type.
// CBOR allows 128 MiB; JSON and unrecognized or missing types use 32 MiB.
// This does not validate the header; adapters must still check content negotiation.
func RequestBodyLimit(contentType string) int64 {
	if MediaTypeOf(contentType) == ContentTypeCbor {
		return MaxCBORRequestBodyBytes
	}
	return MaxJSONRequestBodyBytes
}

// ReadRequestBody reads the Rpc request body up to its encoding-specific limit.
func ReadRequestBody(request *http.Request) ([]byte, error) {
	return ReadBody(request.Body, request.ContentLength, RequestBodyLimit(request.Header.Get(HeaderContentType)), "request")
}

// NewRequest assembles the common HTTP invocation from an encoded invocation envelope.
// Adapters supply identity/auth headers after construction. path begins with '/'.
func NewRequest(ctx context.Context, endpoint, path string, body []byte, contentType, accept string) (*http.Request, error) {
	request, err := http.NewRequestWithContext(ctx, RequestMethod, endpoint+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set(HeaderContentType, contentType)
	request.Header.Set(HeaderAccept, accept)
	EncodeRequestOptionsToHeader(request.Header, ctx)
	return request, nil
}
