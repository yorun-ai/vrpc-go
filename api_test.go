package vrpc_test

import (
	"context"
	"errors"
	"go.yorun.ai/vrpc"
	rpchttp "go.yorun.ai/vrpc/transport/http"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

// Keep the public facade declarative: types are aliases and callable APIs are
// functions, so consumers cannot replace shared behavior through package variables.
func TestFacadeDeclarations(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "api.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, declaration := range file.Decls {
		group, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range group.Specs {
			switch spec := spec.(type) {
			case *ast.TypeSpec:
				if ast.IsExported(spec.Name.Name) && !spec.Assign.IsValid() {
					t.Errorf("%s must alias its internal implementation", spec.Name.Name)
				}
			case *ast.ValueSpec:
				if group.Tok == token.VAR {
					for _, name := range spec.Names {
						if ast.IsExported(name.Name) {
							t.Errorf("%s must not be an exported package variable", name.Name)
						}
					}
				}
			}
		}
	}
}

type userResult struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func TestGenericCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", rpchttp.ContentTypeJson)
		w.Header().Set(rpchttp.HeaderRpcStatus, rpchttp.StatusOK)
		w.Header().Set(rpchttp.HeaderRpcServer, "name=demo.server,version=1.0.0,instanceId=123e4567-e89b-12d3-a456-426614174000")
		switch r.URL.Path {
		case "/demo.Service/user":
			_, _ = io.WriteString(w, `{"result":{"id":9007199254740993,"name":"Ada"}}`)
		case "/demo.Service/list":
			_, _ = io.WriteString(w, `{"result":[1,2,3]}`)
		case "/demo.Service/void":
			_, _ = io.WriteString(w, `{"result":null}`)
		case "/demo.Service/invalid":
			_, _ = io.WriteString(w, `{"result":{"id":42,"name":{}}}`)
		case "/demo.Service/fail":
			w.Header().Set(rpchttp.HeaderRpcStatus, "NOT_FOUND")
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"error":{"code":"NOT_FOUND","message":"missing"}}`)
		}
	}))
	defer server.Close()
	client, err := vrpc.NewClient(vrpc.Option{
		Identity: testClientIdentity(),
		Endpoint: server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	user, _, err := client.Invoke[userResult](t.Context(), methodInfo(t, "user"), nil)
	if err != nil || user.ID != 9007199254740993 || user.Name != "Ada" {
		t.Fatalf("user=%+v err=%v", user, err)
	}
	pointer, metadata, err := client.Invoke[*userResult](t.Context(), methodInfo(t, "user"), nil)
	if err != nil || pointer == nil || *pointer != user || metadata == nil || metadata.HTTPStatus != 200 {
		t.Fatalf("pointer=%+v metadata=%+v err=%v", pointer, metadata, err)
	}
	values, _, err := client.Invoke[[]int](t.Context(), methodInfo(t, "list"), nil)
	if err != nil || !slices.Equal(values, []int{1, 2, 3}) {
		t.Fatalf("values=%v err=%v", values, err)
	}
	if _, _, err := client.Invoke[struct{}](t.Context(), methodInfo(t, "void"), nil); err != nil {
		t.Fatal(err)
	}

	partial, metadata, err := client.Invoke[userResult](t.Context(), methodInfo(t, "invalid"), nil)
	var protocolErr *vrpc.ProtocolError
	if partial != (userResult{}) || metadata == nil || !errors.As(err, &protocolErr) {
		t.Fatalf("partial result leaked: %+v metadata=%+v err=%v", partial, metadata, err)
	}
	missing, metadata, err := client.Invoke[*userResult](t.Context(), methodInfo(t, "fail"), nil)
	var remoteErr *vrpc.InvocationError
	if missing != nil || metadata == nil || metadata.HTTPStatus != 404 || !errors.As(err, &remoteErr) {
		t.Fatalf("remote failure: %+v metadata=%+v err=%v", missing, metadata, err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, metadata, err = client.Invoke[userResult](ctx, methodInfo(t, "user"), nil)
	if !errors.Is(err, context.Canceled) || metadata != nil {
		t.Fatalf("cancellation: metadata=%+v err=%v", metadata, err)
	}
}

func methodInfo(t *testing.T, name string) vrpc.MethodInfo {
	t.Helper()
	registry := vrpc.NewRegistry()
	registry.Register(&vrpc.ServiceSpec{
		SkelName: "demo.Service",
		Methods: []vrpc.MethodSpec{{
			SkelName: name,
		}},
	})
	info, ok := registry.GetMethodInfo("demo.Service", name)
	if !ok {
		t.Fatal("missing method info")
	}
	return info
}

func TestInvokeUsesProvidedMethodInfo(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected := rpchttp.ContentTypeJson
		if calls == 1 {
			expected = rpchttp.ContentTypeCbor
		}
		calls++
		if r.URL.Path != "/demo.Service/get" || r.Header.Get("Content-Type") != expected {
			t.Errorf("method descriptor: path=%s content-type=%s", r.URL.Path, r.Header.Get("Content-Type"))
		}
		w.Header().Set("Content-Type", rpchttp.ContentTypeJson)
		w.Header().Set(rpchttp.HeaderRpcStatus, rpchttp.StatusOK)
		w.Header().Set(rpchttp.HeaderRpcServer, "name=demo.server,version=1.0.0,instanceId=123e4567-e89b-41d4-a716-446655440000")
		_, _ = io.WriteString(w, `{"result":42}`)
	}))
	defer server.Close()
	client, err := vrpc.NewClient(vrpc.Option{
		Identity: testClientIdentity(),
		Endpoint: server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, binary := range []bool{false, true} {
		registry := vrpc.NewRegistry()
		spec := &vrpc.ServiceSpec{
			SkelName: "demo.Service",
			Methods: []vrpc.MethodSpec{{
				SkelName:                    "get",
				ArgumentsContainsBinaryType: binary,
			}},
		}
		registry.Register(spec)
		info, ok := registry.GetMethodInfo("demo.Service", "get")
		if !ok {
			t.Fatal("missing method")
		}
		spec.Methods[0].SkelName = "changed"
		spec.Methods[0].ArgumentsContainsBinaryType = !binary

		result, _, err := client.Invoke[int](t.Context(), info, nil)
		if err != nil || result != 42 {
			t.Fatalf("result=%d err=%v", result, err)
		}
	}
	if _, _, err := client.Invoke[int](t.Context(), vrpc.MethodInfo{}, nil); err == nil {
		t.Fatal("empty method descriptor accepted")
	}
}

func testClientIdentity() vrpc.Identity {
	return vrpc.Identity{
		Name:       "demo.client",
		Version:    "1.0.0",
		InstanceID: "123e4567-e89b-12d3-a456-426614174001",
	}
}

func TestInvokeRaw(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Content-Type") != rpchttp.ContentTypeJson || r.Header.Get("Accept") != rpchttp.ContentTypeJson {
			t.Error("raw invocation must use JSON")
		}
		body, err := io.ReadAll(r.Body)
		if err != nil || string(body) != `{"params":{"id":42}}` {
			t.Errorf("unexpected request: %s, %v", body, err)
		}
		w.Header().Set("Content-Type", rpchttp.ContentTypeJson)
		w.Header().Set(rpchttp.HeaderRpcServer, "name=demo.server,version=1.0.0,instanceId=123e4567-e89b-12d3-a456-426614174000")
		w.Header().Set(rpchttp.HeaderRpcStatus, rpchttp.StatusOK)
		switch r.URL.Path {
		case "/invoke/demo.Raw/Get":
			_, _ = io.WriteString(w, `{"result":{"id":42,"items":[true,"ok"]}}`)
		case "/invoke/demo.Raw/Fail":
			w.Header().Set(rpchttp.HeaderRpcStatus, "NOT_FOUND")
			_, _ = io.WriteString(w, `{"error":{"message":"missing"}}`)
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	client, err := vrpc.NewClient(vrpc.Option{
		Identity: testClientIdentity(),
		Endpoint: server.URL + "/invoke",
	})
	if err != nil {
		t.Fatal(err)
	}
	params := map[string]any{"id": 42}
	result, metadata, err := client.InvokeRaw(t.Context(), "demo.Raw", "Get", params)
	if err != nil || metadata == nil || metadata.Status != rpchttp.StatusOK {
		t.Fatalf("metadata=%+v err=%v", metadata, err)
	}
	object, ok := result.(map[string]any)
	if !ok || object["id"] != float64(42) {
		t.Fatalf("unexpected dynamic result: %#v", result)
	}
	items, ok := object["items"].([]any)
	if !ok || len(items) != 2 || items[0] != true || items[1] != "ok" {
		t.Fatalf("unexpected dynamic array: %#v", object["items"])
	}

	result, metadata, err = client.InvokeRaw(t.Context(), "demo.Raw", "Fail", params)
	var remote *vrpc.InvocationError
	if result != nil || !errors.As(err, &remote) || metadata == nil || remote.Metadata != metadata {
		t.Fatalf("result=%v metadata=%+v err=%v", result, metadata, err)
	}
	for _, method := range []string{"", "bad/path", "bad?query"} {
		result, metadata, err = client.InvokeRaw(t.Context(), "demo.Raw", method, params)
		if result != nil || metadata != nil || err == nil {
			t.Fatalf("invalid method accepted: %q", method)
		}
	}
	if calls != 2 {
		t.Fatalf("unexpected request count: %d", calls)
	}
}
