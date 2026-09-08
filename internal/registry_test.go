package vrpc

import (
	"fmt"
	"sync"
	"testing"
)

func TestRegistrySnapshotAndAtomicRegistration(t *testing.T) {
	var registry Registry
	spec := &ServiceSpec{
		Name:     "Files",
		SkelName: "demo.Files",
		Methods: []MethodSpec{{
			Name:                     "Get",
			SkelName:                 "get",
			ArgumentsSensitive:       true,
			ResultSensitive:          true,
			ResultContainsBinaryType: true,
		}},
	}
	registry.Register(spec)
	spec.Name = "ChangedFiles"
	spec.Methods[0].Name = "ChangedGet"
	spec.Methods[0].SkelName = "changed"
	spec.Methods[0].ArgumentsSensitive = false
	spec.Methods[0].ResultSensitive = false
	spec.Methods[0].ResultContainsBinaryType = false
	info, ok := registry.GetMethodInfo("demo.Files", "get")
	if !ok || info.ServiceSkelName() != "demo.Files" || info.SkelName() != "get" || info.FullURLPath() != "/demo.Files/get" || !info.ResultContainsBinaryType() || info.ArgumentsContainsBinaryType() {
		t.Fatalf("bad snapshot: %+v", info)
	}
	if info.ServiceName() != "Files" || info.Name() != "Get" {
		t.Fatal("input mutation affected Go names")
	}
	if !info.ArgumentsSensitive() || !info.ResultSensitive() {
		t.Fatal("input mutation affected sensitive flags")
	}
	assertRegistrationPanic(t, func() {
		registry.Register(spec)
	})
	if _, ok := registry.GetMethodInfo("demo.Files", "changed"); ok {
		t.Fatal("input mutation affected registry")
	}
	bad := &ServiceSpec{
		SkelName: "demo.Atomic",
		Methods: []MethodSpec{{
			SkelName: "valid",
		}, {
			SkelName: "bad/path",
		}},
	}
	assertRegistrationPanic(t, func() {
		registry.Register(bad)
	})
	if _, ok := registry.GetMethodInfo("demo.Atomic", "valid"); ok {
		t.Fatal("partial registration leaked")
	}
	bad.Methods = bad.Methods[:1]
	registry.Register(bad)
}

func TestRegistryRejectsInvalidContracts(t *testing.T) {
	cases := []*ServiceSpec{
		nil, {
			SkelName: "..",
		}, {
			SkelName: "bad/path",
		},
		{
			SkelName: "demo.Service",
			Methods: []MethodSpec{{
				SkelName: "",
			}},
		},
		{
			SkelName: "demo.Service",
			Methods: []MethodSpec{{
				SkelName: "get",
			}, {
				SkelName: "get",
			}},
		},
	}
	for i, spec := range cases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			assertRegistrationPanic(t, func() {
				NewRegistry().Register(spec)
			})
		})
	}
}

func TestRegistryConcurrentRegistrationAndLookup(t *testing.T) {
	registry := NewRegistry()
	var wg sync.WaitGroup
	for i := range 24 {
		wg.Go(func() {
			name := fmt.Sprintf("demo.Service%d", i)
			registry.Register(&ServiceSpec{
				SkelName: name,
				Methods: []MethodSpec{{
					SkelName: "get",
				}},
			})
			for range 24 {
				if _, ok := registry.GetMethodInfo(name, "get"); !ok {
					t.Error("missing method")
				}
			}
		})
	}
	wg.Wait()
}

func assertRegistrationPanic(t *testing.T, register func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Error("invalid registration did not panic")
		}
	}()
	register()
}
