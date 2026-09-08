package vrpc_test

import (
	"go.yorun.ai/vrpc"
	"testing"
)

func TestIdentityAndTrace(t *testing.T) {
	identity := vrpc.Identity{Name: "demo.client", Version: "1.2.3-beta.1+build.2", InstanceID: "123e4567-e89b-12d3-a456-426614174000"}
	header, err := vrpc.EncodeIdentity(identity)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := vrpc.DecodeIdentity(header)
	if err != nil || decoded != identity {
		t.Fatalf("roundtrip: %v %v", decoded, err)
	}
	for _, version := range []string{"1.2", "1.2.3-01", "01.2.3", "v1.2.3"} {
		identity.Version = version
		if _, err := vrpc.EncodeIdentity(identity); err == nil {
			t.Errorf("accepted version %q", version)
		}
	}
	if _, err := vrpc.DecodeIdentity(header + ",name=other"); err == nil {
		t.Error("duplicate identity field accepted")
	}
	trace := vrpc.NewTrace()
	child := trace.Child()
	if child.ID != trace.ID || child.Span == trace.Span {
		t.Fatal("bad child trace")
	}
	if _, err := vrpc.EncodeTrace(child); err != nil {
		t.Fatal(err)
	}
	if _, err := vrpc.EncodeTrace(vrpc.Trace{ID: "00000000000000000000000000000000", Span: trace.Span}); err == nil {
		t.Error("zero ID accepted")
	}
}
