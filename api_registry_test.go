package vrpc_test

import (
	"fmt"
	"testing"

	"go.yorun.ai/vrpc"
)

func TestMustGetMethodInfo(t *testing.T) {
	var isolated vrpc.Registry
	for _, registry := range []struct {
		name     string
		register func(*vrpc.ServiceSpec)
		get      func(string, string) (vrpc.MethodInfo, bool)
		mustGet  func(string, string) vrpc.MethodInfo
	}{
		{
			name:     "default",
			register: vrpc.Register,
			get:      vrpc.GetMethodInfo,
			mustGet:  vrpc.MustGetMethodInfo,
		},
		{
			name:     "isolated",
			register: isolated.Register,
			get:      isolated.GetMethodInfo,
			mustGet:  isolated.MustGetMethodInfo,
		},
	} {
		t.Run(registry.name, func(t *testing.T) {
			const service = "test.MustLookupService"
			// The default registry persists when tests run with -count.
			if _, ok := registry.get(service, "upload"); !ok {
				registry.register(&vrpc.ServiceSpec{
					Name:     "MustLookupService",
					SkelName: service,
					Methods: []vrpc.MethodSpec{{
						Name:                        "Upload",
						SkelName:                    "upload",
						ArgumentsSensitive:          true,
						ResultSensitive:             true,
						ArgumentsContainsBinaryType: true,
						ResultContainsBinaryType:    true,
					}},
				})
			}
			want, ok := registry.get(service, "upload")
			if !ok {
				t.Fatal("registered method missing")
			}
			if got := registry.mustGet(service, "upload"); got != want {
				t.Fatalf("must lookup changed method metadata: got %+v, want %+v", got, want)
			}
			for _, missing := range [][2]string{{"test.UnregisteredService", "upload"}, {service, "missing"}} {
				t.Run(missing[0]+"/"+missing[1], func(t *testing.T) {
					defer func() {
						value := recover()
						want := "vrpc: method not registered: " + missing[0] + "/" + missing[1]
						if value == nil || fmt.Sprint(value) != want {
							t.Errorf("panic = %v, want %q", value, want)
						}
					}()
					registry.mustGet(missing[0], missing[1])
				})
			}
		})
	}
}
