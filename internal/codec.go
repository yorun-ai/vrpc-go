package vrpc

import (
	"fmt"

	rpchttp "go.yorun.ai/vrpc/transport/http"
)

// _Codec selects built-in JSON (false) or CBOR (true) for one invocation.
type _Codec bool

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
	return rpchttp.EncodeRequest(params, c.ContentType())
}

func (c _Codec) DecodeResponse(data []byte, result any) (*rpchttp.ErrorPayload, error) {
	envelope, err := rpchttp.DecodeResponse(data, c.ContentType())
	if err != nil {
		return nil, err
	}
	return decodePayload(envelope, result)
}

func decodePayload(envelope *rpchttp.ResponsePayload, result any) (*rpchttp.ErrorPayload, error) {
	if len(envelope.ResultBytes) == 0 && len(envelope.ErrorBytes) == 0 {
		return nil, fmt.Errorf("missing response envelope")
	}

	payload, err := envelope.DecodeError()
	if err != nil || payload != nil {
		return payload, err
	}

	if len(envelope.ResultBytes) == 0 {
		return nil, fmt.Errorf("missing result")
	}
	if result != nil {
		return nil, envelope.Unmarshal(envelope.ResultBytes, result)
	}
	return nil, nil
}
