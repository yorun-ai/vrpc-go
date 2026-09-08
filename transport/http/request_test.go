package http

import (
	"bytes"
	"github.com/fxamacker/cbor/v2"
	"net/http"
	"testing"
)

func TestJSONRequestPreservesRawSchemaEncoding(t *testing.T) {
	raw := []byte(`{"legacy":null,"current":[],"id":9007199254740993}`)
	body, err := EncodeJSONRequest(raw)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeJSONRequest(body)
	if err != nil || !bytes.Equal(decoded, raw) {
		t.Fatalf("payload=%x err=%v", decoded, err)
	}
}

func TestCBORRequestPreservesRawSchemaEncoding(t *testing.T) {
	raw, err := cbor.Marshal(map[int64]any{
		9007199254740993: []byte{0, 255},
		1:                nil,
	})
	if err != nil {
		t.Fatal(err)
	}
	body, err := EncodeCBORRequest(raw)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeCBORRequest(body)
	if err != nil || !bytes.Equal(decoded, raw) {
		t.Fatalf("payload=%x err=%v", decoded, err)
	}
}

func TestJSONRequestRequiresParams(t *testing.T) {
	if _, err := DecodeJSONRequest([]byte(`{}`)); err == nil {
		t.Fatal("missing params accepted")
	}
}

func TestReadRequestBodyLimit(t *testing.T) {
	if _, err := ReadRequestBody(&http.Request{
		Body:          http.NoBody,
		ContentLength: MaxRequestBodyBytes + 1,
	}); err == nil {
		t.Fatal("oversized request accepted")
	}
}
