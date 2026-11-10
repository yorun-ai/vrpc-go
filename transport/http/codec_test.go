package http

import (
	"bytes"
	"reflect"
	"testing"
	"uuid"

	skeltype "go.yorun.ai/skel/types"
)

type uuidPayload[T comparable] struct {
	ID       T            `json:"id" cbor:"id"`
	Optional *T           `json:"optional" cbor:"optional"`
	Missing  *T           `json:"missing" cbor:"missing"`
	IDs      []T          `json:"ids" cbor:"ids"`
	Lookup   map[T]string `json:"lookup" cbor:"lookup"`
}

func TestCodecNativeUUIDCompatibility(t *testing.T) {
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	legacyID := skeltype.NewUUID(id)
	native := uuidPayload[uuid.UUID]{
		ID:       id,
		Optional: &id,
		IDs:      []uuid.UUID{id, {}},
		Lookup:   map[uuid.UUID]string{id: "value"},
	}
	legacy := uuidPayload[skeltype.UUID]{
		ID:       legacyID,
		Optional: &legacyID,
		IDs:      []skeltype.UUID{legacyID, {}},
		Lookup:   map[skeltype.UUID]string{legacyID: "value"},
	}
	for _, contentType := range []string{ContentTypeJson, ContentTypeCbor} {
		t.Run(contentType, func(t *testing.T) {
			oldRequest, err := EncodeRequest(legacy, contentType)
			if err != nil {
				t.Fatal(err)
			}
			request, err := EncodeRequest(native, contentType)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(oldRequest, request) {
				t.Fatalf("request encoding changed: %x != %x", oldRequest, request)
			}
			var decoded uuidPayload[uuid.UUID]
			if err := DecodeRequest(oldRequest, &decoded, contentType); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(native, decoded) {
				t.Fatalf("request mismatch: %#v", decoded)
			}
			response, err := EncodeResponse(native, nil, contentType)
			if err != nil {
				t.Fatal(err)
			}
			oldResponse, err := EncodeResponse(legacy, nil, contentType)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(oldResponse, response) {
				t.Fatalf("response encoding changed: %x != %x", oldResponse, response)
			}
			envelope, err := DecodeResponse(response, contentType)
			if err != nil {
				t.Fatal(err)
			}
			var oldDecoded uuidPayload[skeltype.UUID]
			if err := envelope.Unmarshal(envelope.ResultBytes, &oldDecoded); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(legacy, oldDecoded) {
				t.Fatalf("legacy response mismatch: %#v", oldDecoded)
			}
			decoded = uuidPayload[uuid.UUID]{}
			if err := envelope.Unmarshal(envelope.ResultBytes, &decoded); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(native, decoded) {
				t.Fatalf("response mismatch: %#v", decoded)
			}
		})
	}
}
