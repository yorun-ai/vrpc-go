package vrpc

import (
	"encoding/json/v2"
	"fmt"
	wire "go.yorun.ai/vrpc/transport/http"
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
	encoded, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	return wire.EncodeJSONRequest(encoded)
}

// DecodeResponse decodes the vRPC envelope and an optional typed result.
func (JSONCodec) DecodeResponse(data []byte, result any) (*ErrorPayload, error) {
	envelope, err := wire.DecodeJSONResponse(data)
	if err != nil {
		return nil, err
	}
	return decodePayload(envelope, result)
}

func decodePayload(envelope *wire.ResponsePayload, result any) (*ErrorPayload, error) {
	if len(envelope.ResultBytes) == 0 && len(envelope.ErrorBytes) == 0 {
		return nil, fmt.Errorf("missing response envelope")
	}
	if !wire.IsEmptyErrorPayload(envelope.ErrorBytes) {
		var payload ErrorPayload
		if err := envelope.Unmarshal(envelope.ErrorBytes, &payload); err != nil {
			return nil, err
		}
		return &payload, nil
	}
	if len(envelope.ResultBytes) == 0 {
		return nil, fmt.Errorf("missing result")
	}
	if result != nil {
		return nil, envelope.Unmarshal(envelope.ResultBytes, result)
	}
	return nil, nil
}
