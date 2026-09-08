package http

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type closeBody struct {
	io.Reader
	closed bool
}

func (b *closeBody) Close() error { b.closed = true; return nil }
func TestExchangeOwnsBodyAndPreservesFailureStage(t *testing.T) {
	request, err := NewRequest(t.Context(), "http://example.com/invoke", "/demo.Service/Get", []byte(`{"params":{}}`), ContentTypeJson, ContentTypeJson)
	if err != nil {
		t.Fatal(err)
	}
	for _, failDecode := range []bool{false, true} {
		body := &closeBody{Reader: strings.NewReader(`{"result":42}`)}
		prepared := false
		calls := 0
		cause := errors.New("decode failed")
		value, err := Exchange(request, func() { prepared = true }, func(*http.Request) (*http.Response, error) {
			calls++
			if !prepared {
				t.Error("request was not prepared")
			}
			return &http.Response{Body: body}, nil
		}, func(response *http.Response) (int, error) {
			if failDecode {
				return 7, cause
			}
			return 42, nil
		})
		if !body.closed || calls != 1 {
			t.Fatalf("closed=%v calls=%d", body.closed, calls)
		}
		if failDecode {
			var decoded *DecodeError
			if !errors.As(err, &decoded) || !errors.Is(err, cause) || value != 7 {
				t.Fatalf("%d %v", value, err)
			}
		} else if err != nil || value != 42 {
			t.Fatalf("%d %v", value, err)
		}
	}
	_, err = Exchange(request, nil, func(*http.Request) (*http.Response, error) { return nil, context.Canceled }, func(*http.Response) (int, error) { t.Fatal("decoded failed exchange"); return 0, nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
