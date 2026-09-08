// Package cbor adds optional CBOR and Binary support to the vRPC client.
package cbor

import (
	"fmt"

	wire "github.com/fxamacker/cbor/v2"
	"go.yorun.ai/vrpc"
)

// Codec encodes binary values as CBOR byte strings and nil collections as empty.
// Use struct cbor tags (or matching json tags) to specify wire field names.
type Codec struct{}

var encodeMode = func() wire.EncMode {
	mode, err := (wire.EncOptions{NilContainers: wire.NilContainerAsEmpty}).EncMode()
	if err != nil {
		panic(err)
	}
	return mode
}()
var decodeMode = func() wire.DecMode {
	mode, err := (wire.DecOptions{DupMapKey: wire.DupMapKeyEnforcedAPF}).DecMode()
	if err != nil {
		panic(err)
	}
	return mode
}()

// ContentType returns the vRPC media type.
func (Codec) ContentType() string { return vrpc.ContentTypeCBOR }

// EncodeRequest wraps params in a vRPC request envelope.
func (Codec) EncodeRequest(params any) ([]byte, error) {
	if params == nil {
		params = struct{}{}
	}
	return encodeMode.Marshal(struct {
		Params any `cbor:"params"`
	}{params})
}

// DecodeResponse decodes the vRPC envelope and an optional typed result.
func (Codec) DecodeResponse(data []byte, result any) (*vrpc.ErrorPayload, error) {
	var envelope struct {
		Result wire.RawMessage `cbor:"result"`
		Error  wire.RawMessage `cbor:"error"`
	}
	if err := decodeMode.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}
	if len(envelope.Result) == 0 && len(envelope.Error) == 0 {
		return nil, fmt.Errorf("missing response envelope")
	}
	if len(envelope.Error) > 0 && !(len(envelope.Error) == 1 && envelope.Error[0] == 0xf6) {
		var payload vrpc.ErrorPayload
		if err := decodeMode.Unmarshal(envelope.Error, &payload); err != nil {
			return nil, err
		}
		return &payload, nil
	}
	if len(envelope.Result) == 0 {
		return nil, fmt.Errorf("missing result")
	}
	if result != nil {
		return nil, decodeMode.Unmarshal(envelope.Result, result)
	}
	return nil, nil
}
