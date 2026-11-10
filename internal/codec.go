package vrpc

import (
	"fmt"

	rpchttp "go.yorun.ai/vrpc/transport/http"
)

func encodeRequest(params any, contentType string) ([]byte, error) {
	if params == nil {
		params = struct{}{}
	}
	return rpchttp.EncodeRequest(params, contentType)
}

func decodeResponsePayload(data []byte, result any, contentType string) (*rpchttp.ErrorPayload, error) {
	envelope, err := rpchttp.DecodeResponse(data, contentType)
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
