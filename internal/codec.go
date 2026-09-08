package vrpc

import (
	"encoding/json/v2"
	"fmt"

	"github.com/fxamacker/cbor/v2"
	rpchttp "go.yorun.ai/vrpc/transport/http"
)

// _Codec selects built-in JSON (false) or CBOR (true) for one invocation.
type _Codec bool

var (
	encodeMode cbor.EncMode
	decodeMode cbor.DecMode
)

func init() {
	var err error
	encodeMode, err = (cbor.EncOptions{
		NilContainers: cbor.NilContainerAsEmpty,
	}).EncMode()
	if err != nil {
		panic(err)
	}

	decodeMode, err = (cbor.DecOptions{
		DupMapKey: cbor.DupMapKeyEnforcedAPF,
	}).DecMode()
	if err != nil {
		panic(err)
	}
}

func (c _Codec) ContentType() string {
	if c {
		return rpchttp.ContentTypeCbor
	}
	return rpchttp.ContentTypeJson
}

func (c _Codec) EncodeRequest(params any) ([]byte, error) {
	if params == nil {
		params = struct{}{}
	}
	if c {
		raw, err := encodeMode.Marshal(params)
		if err != nil {
			return nil, err
		}
		return rpchttp.EncodeCBORRequest(raw)
	}

	raw, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	return rpchttp.EncodeJSONRequest(raw)
}

func (c _Codec) DecodeResponse(data []byte, result any) (*ErrorPayload, error) {
	var envelope *rpchttp.ResponsePayload
	var err error
	if c {
		envelope, err = rpchttp.DecodeCBORResponseWith(data, decodeMode.Unmarshal)
	} else {
		envelope, err = rpchttp.DecodeJSONResponse(data)
	}
	if err != nil {
		return nil, err
	}
	return decodePayload(envelope, result)
}

func decodePayload(envelope *rpchttp.ResponsePayload, result any) (*ErrorPayload, error) {
	if len(envelope.ResultBytes) == 0 && len(envelope.ErrorBytes) == 0 {
		return nil, fmt.Errorf("missing response envelope")
	}

	if !rpchttp.IsEmptyErrorPayload(envelope.ErrorBytes) {
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
