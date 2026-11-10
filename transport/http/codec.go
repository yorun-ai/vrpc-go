package http

import (
	"fmt"

	"go.yorun.ai/skel/types/codec"
)

// _Encoding identifies the payload and envelope format.
type _Encoding uint8

const (
	encodingJSON _Encoding = iota
	encodingCBOR
)

func encodingForContentType(contentType string) (_Encoding, error) {
	switch MediaTypeOf(contentType) {
	case ContentTypeJson:
		return encodingJSON, nil
	case ContentTypeCbor:
		return encodingCBOR, nil
	default:
		return encodingJSON, fmt.Errorf("unsupported rpc content type %q", contentType)
	}
}

func (e _Encoding) marshal(value any) ([]byte, error) {
	if e == encodingCBOR {
		return codec.MarshalCBOR(value)
	}
	return codec.MarshalJSON(value)
}

func (e _Encoding) unmarshal(data []byte, target any) error {
	if e == encodingCBOR {
		return codec.UnmarshalCBOR(data, target)
	}
	return codec.UnmarshalJSON(data, target)
}
