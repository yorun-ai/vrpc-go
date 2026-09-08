package http

import (
	"errors"
	"strings"
	"testing"
)

func TestRequestContentTypes(t *testing.T) {
	for _, tt := range []struct {
		arguments, result   bool
		contentType, accept string
	}{
		{false, false, ContentTypeJson, ContentTypeJson},
		{true, false, ContentTypeCbor, ContentTypeJson},
		{false, true, ContentTypeJson, ContentTypeCbor + ", " + ContentTypeJson},
		{true, true, ContentTypeCbor, ContentTypeCbor + ", " + ContentTypeJson},
	} {
		contentType, accept := RequestContentTypes(tt.arguments, tt.result)
		if contentType != tt.contentType || accept != tt.accept {
			t.Errorf("flags %v/%v: %q, %q", tt.arguments, tt.result, contentType, accept)
		}
	}
}

func TestReadBodyAcceptsBodyAtLimit(t *testing.T) {
	body, err := ReadBody(strings.NewReader("12345678"), -1, 8, "request")
	if err != nil {
		t.Fatalf("ReadBody() error = %v", err)
	}
	if string(body) != "12345678" {
		t.Fatalf("ReadBody() = %q", body)
	}
}

func TestReadBodyRejectsUnknownLengthBodyOverLimit(t *testing.T) {
	_, err := ReadBody(strings.NewReader("123456789"), -1, 8, "request")
	if err == nil || err.Error() != "rpc request body exceeds 8 byte limit" {
		t.Fatalf("ReadBody() error = %v", err)
	}
}

func TestReadBodyRejectsDeclaredLengthBeforeReading(t *testing.T) {
	reader := &failReader{}
	_, err := ReadBody(reader, 9, 8, "response")
	if err == nil || err.Error() != "rpc response body exceeds 8 byte limit" {
		t.Fatalf("ReadBody() error = %v", err)
	}
	if reader.read {
		t.Fatal("ReadBody() read a body with an oversized declared length")
	}
}

func TestReadBodyReturnsReadError(t *testing.T) {
	want := errors.New("read failed")
	_, err := ReadBody(&errorReader{
		err: want,
	}, -1, 8, "request")
	if !errors.Is(err, want) {
		t.Fatalf("ReadBody() error = %v, want %v", err, want)
	}
}

type failReader struct {
	read bool
}

func (r *failReader) Read([]byte) (int, error) {
	r.read = true
	return 0, errors.New("unexpected read")
}

type errorReader struct {
	err error
}

func (r *errorReader) Read([]byte) (int, error) {
	return 0, r.err
}
