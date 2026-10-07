package http

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/fxamacker/cbor/v2"
)

func TestJSONRequestPreservesEncodedParams(t *testing.T) {
	raw := []byte(`{"nullable":null,"items":[],"id":9007199254740993}`)
	body, err := EncodeJSONRequest(raw)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeJSONRequest(body)
	if err != nil || !bytes.Equal(decoded, raw) {
		t.Fatalf("payload=%x err=%v", decoded, err)
	}
}

func TestCBORRequestPreservesEncodedParams(t *testing.T) {
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

func TestCBORRequestRejectsDuplicateParams(t *testing.T) {
	// {"params": {}, "params": {}}
	body := []byte("\xa2\x66params\xa0\x66params\xa0")
	if params, err := DecodeCBORRequest(body); err == nil || params != nil {
		t.Fatalf("duplicate params accepted: %x, %v", params, err)
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

func TestRequestCodecPreservesValuesAndNormalizesCollections(t *testing.T) {
	type params struct {
		ID       int64             `json:"id"`
		Items    []string          `json:"items"`
		Labels   map[string]string `json:"labels"`
		Optional *int64            `json:"optional"`
	}
	for _, contentType := range []string{ContentTypeJson, ContentTypeCbor} {
		t.Run(contentType, func(t *testing.T) {
			body, err := EncodeRequest(params{
				ID: 9007199254740993,
			}, contentType)
			if err != nil {
				t.Fatal(err)
			}
			var decoded params
			if err := DecodeRequest(body, &decoded, contentType); err != nil {
				t.Fatal(err)
			}
			if decoded.ID != 9007199254740993 || decoded.Items == nil || decoded.Labels == nil || decoded.Optional != nil {
				t.Fatalf("unexpected wire values: %+v", decoded)
			}
		})
	}
}
