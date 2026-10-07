package http

import (
	"bytes"
	"errors"
	"net/http"
	"testing"

	"github.com/fxamacker/cbor/v2"
)

func TestCBORResponseRejectsDuplicateKeys(t *testing.T) {
	// {"result": null, "result": null}
	if payload, err := DecodeCBORResponse([]byte("\xa2\x66result\xf6\x66result\xf6")); err == nil || payload != nil {
		t.Fatalf("duplicate result accepted: %+v, %v", payload, err)
	}

	// Raw result/error values are decoded after the envelope: {"id": 1, "id": 2}.
	raw := []byte("\xa2\x62id\x01\x62id\x02")
	for _, field := range []string{"result", "error"} {
		t.Run(field, func(t *testing.T) {
			body, err := cbor.Marshal(map[string]cbor.RawMessage{
				field: raw,
			})
			if err != nil {
				t.Fatal(err)
			}
			payload, err := DecodeCBORResponse(body)
			if err != nil {
				t.Fatal(err)
			}
			data := payload.ResultBytes
			if field == "error" {
				data = payload.ErrorBytes
			}
			var value map[string]int
			var duplicate *cbor.DupMapKeyError
			if err := payload.Unmarshal(data, &value); !errors.As(err, &duplicate) {
				t.Fatalf("expected duplicate key error, got %v", err)
			}
		})
	}
}

func TestJSONResponsePreservesEncodedResult(t *testing.T) {
	raw := []byte(`{"nullable":null,"items":[],"id":9007199254740993}`)
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

func TestCBORResponsePreservesEncodedResult(t *testing.T) {
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

func TestResponseCodecPreservesResultWhenReplacingError(t *testing.T) {
	type result struct {
		ID    int64    `json:"id"`
		Items []string `json:"items"`
	}
	for _, contentType := range []string{ContentTypeJson, ContentTypeCbor} {
		t.Run(contentType, func(t *testing.T) {
			wantError := ErrorPayload{
				Code:    "OPERATION_FAILED",
				Message: "write failed",
				Reason:  "quota-exceeded",
				Detail:  "private",
			}
			body, err := EncodeResponse(result{
				ID: 9007199254740993,
			}, &wantError, contentType)
			if err != nil {
				t.Fatal(err)
			}
			payload, err := DecodeResponse(body, contentType)
			if err != nil {
				t.Fatal(err)
			}
			var decoded result
			if err := payload.Unmarshal(payload.ResultBytes, &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded.ID != 9007199254740993 || decoded.Items == nil {
				t.Fatalf("unexpected result: %+v", decoded)
			}
			errorValue, err := payload.DecodeError()
			if err != nil || errorValue == nil || *errorValue != wantError {
				t.Fatalf("unexpected error: %+v, %v", errorValue, err)
			}
			errorValue.Detail = ""
			body, err = payload.EncodeWithError(errorValue)
			if err != nil {
				t.Fatal(err)
			}
			rewritten, err := DecodeResponse(body, contentType)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(payload.ResultBytes, rewritten.ResultBytes) {
				t.Fatal("replacing an error changed the result bytes")
			}
			wantError.Detail = ""
			errorValue, err = rewritten.DecodeError()
			if err != nil || errorValue == nil || *errorValue != wantError {
				t.Fatalf("unexpected rewritten error: %+v, %v", errorValue, err)
			}
		})
	}
}
