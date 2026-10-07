package http

import (
	"encoding/json/v2"
	"fmt"

	"github.com/fxamacker/cbor/v2"
)

// _Codec selects the fixed JSON or CBOR wire rules.
type _Codec bool

var cborEncodeMode cbor.EncMode
var cborDecodeMode cbor.DecMode

func init() {
	var err error
	cborEncodeMode, err = (cbor.EncOptions{
		NilContainers: cbor.NilContainerAsEmpty,
	}).EncMode()
	if err != nil {
		panic(err)
	}

	cborDecodeMode, err = (cbor.DecOptions{
		DupMapKey: cbor.DupMapKeyEnforcedAPF,
	}).DecMode()
	if err != nil {
		panic(err)
	}
}

func codecForContentType(contentType string) (_Codec, error) {
	switch MediaTypeOf(contentType) {
	case ContentTypeJson:
		return false, nil
	case ContentTypeCbor:
		return true, nil
	default:
		return false, fmt.Errorf("unsupported rpc content type %q", contentType)
	}
}

func (c _Codec) marshal(value any) ([]byte, error) {
	if c {
		return cborEncodeMode.Marshal(value)
	}
	return json.Marshal(value)
}

func (c _Codec) unmarshal(data []byte, target any) error {
	if c {
		return cborDecodeMode.Unmarshal(data, target)
	}
	return json.Unmarshal(data, target)
}
