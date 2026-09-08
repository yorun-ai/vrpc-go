package http

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"net/http"

	"github.com/fxamacker/cbor/v2"
)

// JSONRequest is the raw vRPC JSON request envelope.
type JSONRequest struct {
	Params jsontext.Value `json:"params"`
}

// EncodeJSONRequest wraps already-encoded arguments without changing schema encoding.
func EncodeJSONRequest(params []byte) ([]byte, error) {
	return json.Marshal(&JSONRequest{
		Params: params,
	})
}

// DecodeJSONRequest extracts encoded arguments from a JSON request envelope.
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

// CBORRequest is the raw vRPC CBOR request envelope.
type CBORRequest struct {
	Params cbor.RawMessage `json:"params"`
}

// EncodeCBORRequest wraps schema-encoded CBOR parameters without modifying their representation.
func EncodeCBORRequest(params []byte) ([]byte, error) {
	return cbor.Marshal(&CBORRequest{
		Params: params,
	})
}

// DecodeCBORRequest extracts raw parameters from a CBOR request envelope.
func DecodeCBORRequest(body []byte) ([]byte, error) {
	var payload CBORRequest
	if err := cbor.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("request body cannot be parsed")
	}
	if len(payload.Params) == 0 {
		return nil, fmt.Errorf("missing request params")
	}
	return payload.Params, nil
}

// ReadRequestBody reads the Rpc request body up to the fixed transport limit.
func ReadRequestBody(request *http.Request) ([]byte, error) {
	return ReadBody(request.Body, request.ContentLength, MaxRequestBodyBytes, "request")
}

// NewRequest assembles the common HTTP invocation from a schema-encoded envelope.
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
