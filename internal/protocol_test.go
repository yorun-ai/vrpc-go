package vrpc

import (
	"testing"
)

func TestIdentityAndTrace(t *testing.T) {
	identity := Identity{
		Name:       "demo.client",
		Version:    "1.2.3-beta.1+build.2",
		InstanceID: "123e4567-e89b-12d3-a456-426614174000",
	}
	header, err := EncodeIdentity(identity)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeIdentity(header)
	if err != nil || decoded != identity {
		t.Fatalf("roundtrip: %v %v", decoded, err)
	}
	for _, version := range []string{"1.2", "1.2.3-01", "01.2.3", "v1.2.3"} {
		identity.Version = version
		if _, err := EncodeIdentity(identity); err == nil {
			t.Errorf("accepted version %q", version)
		}
	}
	if _, err := DecodeIdentity(header + ",name=other"); err == nil {
		t.Error("duplicate identity field accepted")
	}
	trace := NewTrace()
	child := trace.Child()
	if child.ID != trace.ID || child.Span == trace.Span {
		t.Fatal("bad child trace")
	}
	if _, err := EncodeTrace(child); err != nil {
		t.Fatal(err)
	}
	if _, err := EncodeTrace(Trace{
		ID:   "00000000000000000000000000000000",
		Span: trace.Span,
	}); err == nil {
		t.Error("zero ID accepted")
	}
}
