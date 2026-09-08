package http

import (
	"bytes"
	"github.com/fxamacker/cbor/v2"
	"net/http"
	"testing"
)

func TestJSONResponsePreservesRawSchemaEncoding(t *testing.T) {
	raw := []byte(`{"legacy":null,"current":[],"id":9007199254740993}`)
	body, err := EncodeJSONResponse(raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeJSONResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decoded.ResultBytes, raw) || !IsEmptyErrorPayload(decoded.ErrorBytes) {
		t.Fatalf("payload=%+v", decoded)
	}
}

func TestCBORResponsePreservesRawSchemaEncoding(t *testing.T) {
	raw, err := cbor.Marshal(map[int64]any{
		9007199254740993: []byte{0, 255},
		1:                nil,
	})
	if err != nil {
		t.Fatal(err)
	}
	body, err := EncodeCBORResponse(raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeCBORResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decoded.ResultBytes, raw) || !IsEmptyErrorPayload(decoded.ErrorBytes) {
		t.Fatalf("payload=%+v", decoded)
	}
}

func TestReadResponseBodyLimit(t *testing.T) {
	if _, err := ReadResponseBody(&http.Response{
		Body:          http.NoBody,
		ContentLength: MaxResponseBodyBytes + 1,
	}); err == nil {
		t.Fatal("oversized response accepted")
	}
}
