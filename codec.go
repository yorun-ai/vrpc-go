package vrpc

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
)

// Codec encodes request envelopes and decodes response envelopes.
// DecodeResponse returns the remote error payload when present; otherwise it
// decodes result into the supplied pointer. A nil result discards the value.
// Implementations must reject malformed envelopes and be safe for concurrent use.
type Codec interface {
	ContentType() string
	EncodeRequest(params any) ([]byte, error)
	DecodeResponse(data []byte, result any) (*ErrorPayload, error)
}

// JSONCodec implements vRPC JSON using encoding/json/v2. Nil slices and maps
// encode as empty collections, matching current Vine collection semantics.
type JSONCodec struct{}

// ContentType returns the vRPC media type.
func (JSONCodec) ContentType() string { return ContentTypeJSON }

// EncodeRequest wraps params in a vRPC request envelope.
func (JSONCodec) EncodeRequest(params any) ([]byte, error) {
	if params == nil {
		params = struct{}{}
	}
	return json.Marshal(struct {
		Params any `json:"params"`
	}{params})
}

// DecodeResponse decodes the vRPC envelope and an optional typed result.
func (JSONCodec) DecodeResponse(data []byte, result any) (*ErrorPayload, error) {
	var envelope struct {
		Result jsontext.Value `json:"result"`
		Error  jsontext.Value `json:"error"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}
	if len(envelope.Result) == 0 && len(envelope.Error) == 0 {
		return nil, fmt.Errorf("missing response envelope")
	}
	if len(envelope.Error) > 0 && !bytes.Equal(bytes.TrimSpace(envelope.Error), []byte("null")) {
		var payload ErrorPayload
		if err := json.Unmarshal(envelope.Error, &payload); err != nil {
			return nil, err
		}
		return &payload, nil
	}
	if len(envelope.Result) == 0 {
		return nil, fmt.Errorf("missing result")
	}
	if result != nil {
		return nil, json.Unmarshal(envelope.Result, result)
	}
	return nil, nil
}
