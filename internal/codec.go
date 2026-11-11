package vrpc

import (
	"fmt"
	"mime"
	"net/http"

	rpchttp "go.yorun.ai/vrpc/transport/http"
)

func encodeRequest(params any, contentType string) ([]byte, error) {
	if params == nil {
		params = struct{}{}
	}
	return rpchttp.EncodeRequest(params, contentType)
}

func decodeResponse(response *http.Response, metadata *ResponseMetadata, result any, encoding _InvokeEncoding) (*rpchttp.ErrorPayload, error) {
	if err := rpchttp.CheckResponseHeaders(response.Header); err != nil {
		return nil, err
	}
	if !statusPattern.MatchString(metadata.Status) {
		return nil, fmt.Errorf("invalid status")
	}

	server, err := DecodeIdentity(response.Header.Get(rpchttp.HeaderRpcServer))
	if err != nil {
		return nil, err
	}
	metadata.Server = server

	contentType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}
	if contentType != rpchttp.ContentTypeJson {
		if !encoding.resultContainsBinary || contentType != rpchttp.ContentTypeCbor {
			return nil, fmt.Errorf("unsupported response content type %s", contentType)
		}
	}

	if encoding := response.Header.Get("Content-Encoding"); encoding != "" && encoding != "identity" {
		return nil, fmt.Errorf("unsupported response content encoding %s", encoding)
	}

	body, err := rpchttp.ReadResponseBody(response)
	if err != nil {
		return nil, err
	}

	if metadata.Status != rpchttp.StatusOK || metadata.HTTPStatus < 200 || metadata.HTTPStatus >= 300 {
		result = nil
	}

	return decodeResponsePayload(body, result, contentType)
}

func decodeResponsePayload(data []byte, result any, contentType string) (*rpchttp.ErrorPayload, error) {
	envelope, err := rpchttp.DecodeResponse(data, contentType)
	if err != nil {
		return nil, err
	}

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
