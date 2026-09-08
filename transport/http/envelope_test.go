package http

import (
	"bytes"
	"testing"
)

func TestJSONEnvelopesPreserveSchemaEncodedPayloads(t *testing.T) {
	// Null legacy collections and integers beyond float64 precision must not be normalized.
	payload := []byte(`{"legacy":null,"current":[],"id":9007199254740993}`)
	request, err := EncodeJSONRequest(payload)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeJSONRequest(request)
	if err != nil || !bytes.Equal(decoded, payload) {
		t.Fatalf("%s %v", decoded, err)
	}
	response, err := EncodeJSONResponse(payload, nil)
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := DecodeJSONResponse(response)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(envelope.ResultBytes, payload) || !IsEmptyErrorPayload(envelope.ErrorBytes) {
		t.Fatalf("%+v", envelope)
	}
	if _, err := DecodeJSONRequest([]byte(`{}`)); err == nil {
		t.Error("missing params accepted")
	}
}
