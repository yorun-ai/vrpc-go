package http

import (
	"bytes"
	"fmt"
	"io"
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
		ContentLength: MaxJSONRequestBodyBytes + 1,
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

func TestRequestBodyLimitByContentType(t *testing.T) {
	for _, tt := range []struct {
		contentType string
		want        int64
	}{
		{
			contentType: ContentTypeJson,
			want:        32 << 20,
		},
		{
			contentType: ContentTypeCbor,
			want:        128 << 20,
		},
		{
			contentType: "Application/VRPC+CBOR; charset=binary",
			want:        128 << 20,
		},
		{
			contentType: "",
			want:        32 << 20,
		},
		{
			contentType: "application/cbor",
			want:        32 << 20,
		},
	} {
		t.Run(tt.contentType, func(t *testing.T) {
			if got := RequestBodyLimit(tt.contentType); got != tt.want {
				t.Fatalf("limit = %d, want %d", got, tt.want)
			}
			reader := &failReader{}
			request := &http.Request{
				Header:        http.Header{},
				Body:          io.NopCloser(reader),
				ContentLength: tt.want + 1,
			}
			request.Header.Set(HeaderContentType, tt.contentType)
			_, err := ReadRequestBody(request)
			wantError := fmt.Sprintf("rpc request body exceeds %d byte limit", tt.want)
			if err == nil || err.Error() != wantError || reader.read {
				t.Fatalf("oversized declared length: err=%v read=%v", err, reader.read)
			}
			request.Body = http.NoBody
			request.ContentLength = tt.want
			if _, err := ReadRequestBody(request); err != nil {
				t.Fatalf("declared length at limit: %v", err)
			}
		})
	}
	if MaxJSONRequestBodyBytes != 32<<20 || MaxResponseBodyBytes != 128<<20 {
		t.Fatal("JSON request or response limit changed")
	}
}

func TestReadRequestBodyUnknownLengthBoundaries(t *testing.T) {
	for _, tt := range []struct {
		contentType string
		limit       int64
	}{
		{
			contentType: ContentTypeJson,
			limit:       32 << 20,
		},
		{
			contentType: ContentTypeCbor + "; charset=binary",
			limit:       128 << 20,
		},
	} {
		t.Run(tt.contentType, func(t *testing.T) {
			for _, extra := range []int64{0, 1} {
				t.Run(fmt.Sprint(extra), func(t *testing.T) {
					request := &http.Request{
						Header:           http.Header{},
						Body:             io.NopCloser(io.LimitReader(zeroBodyReader{}, tt.limit+extra)),
						ContentLength:    -1,
						TransferEncoding: []string{"chunked"},
					}
					request.Header.Set(HeaderContentType, tt.contentType)
					body, err := ReadRequestBody(request)
					if extra == 0 {
						if err != nil || int64(len(body)) != tt.limit {
							t.Fatalf("body at limit: length=%d err=%v", len(body), err)
						}
					} else {
						want := fmt.Sprintf("rpc request body exceeds %d byte limit", tt.limit)
						if err == nil || err.Error() != want || body != nil {
							t.Fatalf("oversized body: length=%d err=%v", len(body), err)
						}
					}
				})
			}
		})
	}
}

type zeroBodyReader struct{}

func (zeroBodyReader) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}
