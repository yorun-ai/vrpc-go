// Package cbor adds optional CBOR and Binary support to the vRPC client.
package cbor

import (
	"fmt"

	wire "github.com/fxamacker/cbor/v2"
	"go.yorun.ai/vrpc"
	transport "go.yorun.ai/vrpc/transport/http"
	cborwire "go.yorun.ai/vrpc/transport/http/cbor"
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
	encoded, err := encodeMode.Marshal(params)
	if err != nil {
		return nil, err
	}
	return cborwire.EncodeRequest(encoded)
}

// DecodeResponse decodes the vRPC envelope and an optional typed result.
func (Codec) DecodeResponse(data []byte, result any) (*vrpc.ErrorPayload, error) {
	envelope, err := cborwire.DecodeResponseWith(data, decodeMode.Unmarshal)
	if err != nil {
		return nil, err
	}
	if len(envelope.ResultBytes) == 0 && len(envelope.ErrorBytes) == 0 {
		return nil, fmt.Errorf("missing response envelope")
	}
	if !transport.IsEmptyErrorPayload(envelope.ErrorBytes) {
		var payload vrpc.ErrorPayload
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
