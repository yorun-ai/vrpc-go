package cbor

import (
	"bytes"
	"github.com/fxamacker/cbor/v2"
	"testing"
)

func TestCBORPreservesRawSchemaEncoding(t *testing.T) {
	// Preserve a byte string, integer map key and null, without decoding via any/JSON.
	raw, err := cbor.Marshal(map[int64]any{9007199254740993: []byte{0, 255}, 1: nil})
	if err != nil {
		t.Fatal(err)
	}
	request, err := EncodeRequest(raw)
	if err != nil {
		t.Fatal(err)
	}
	params, err := DecodeRequest(request)
	if err != nil || !bytes.Equal(params, raw) {
		t.Fatalf("%x %v", params, err)
	}
	response, err := EncodeResponse(raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeResponse(response)
	if err != nil || !bytes.Equal(decoded.ResultBytes, raw) {
		t.Fatalf("%+v %v", decoded, err)
	}
}
