package vrpc

import (
	"compress/gzip"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
	rpchttp "go.yorun.ai/vrpc/transport/http"
)

const serverIdentity = "name=demo.server,version=1.0.0,instanceId=123e4567-e89b-12d3-a456-426614174000"

func reply(w http.ResponseWriter, status string, httpStatus int, body string) {
	w.Header().Set("Content-Type", rpchttp.ContentTypeJson)
	w.Header().Set(rpchttp.HeaderRpcStatus, status)
	w.Header().Set(rpchttp.HeaderRpcServer, serverIdentity)
	w.WriteHeader(httpStatus)
	_, _ = io.WriteString(w, body)
}
func newClient(t *testing.T, options Option) *Client {
	t.Helper()
	c, err := NewClient(options)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func TestCallWireAndIdentity(t *testing.T) {
	parent := NewTrace()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/prefix/invoke/demo.Service/Get" {
			t.Errorf("route: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "key token" {
			t.Error("missing credentials")
		}
		if r.Header.Get("vrpc-actor") != "" || r.Header.Get("vrpc-initiator") != "" {
			t.Error("unexpected framework identity")
		}
		if _, err := DecodeIdentity(r.Header.Get(rpchttp.HeaderRpcClient)); err != nil {
			t.Error(err)
		}
		if !strings.Contains(r.Header.Get(rpchttp.HeaderRpcTrace), parent.ID) || strings.Contains(r.Header.Get(rpchttp.HeaderRpcTrace), parent.Span) {
			t.Error("trace not derived")
		}
		if !strings.HasPrefix(r.Header.Get(rpchttp.HeaderRpcOptions), "timeout=") {
			t.Error("missing timeout")
		}
		var body struct {
			Params struct {
				ID int64 `json:"id"`
			} `json:"params"`
		}
		bytes, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(bytes, &body); err != nil || body.Params.ID != 9007199254740993 {
			t.Errorf("params: %s %v", bytes, err)
		}
		w.Header().Set(rpchttp.HeaderPortalTraceID, parent.ID)
		reply(w, "OK", 200, `{"result":{"id":9007199254740993,"names":[]},"error":null}`)
	}))
	defer server.Close()
	headers := http.Header{
		"X-Example": []string{"original"},
	}
	headers.Set("Authorization", "key token")
	client := newClient(t, Option{
		Identity: testClientIdentity(),
		Endpoint: server.URL + "/prefix/invoke/",
		Headers:  headers,
	})

	headers.Set("X-Example", "mutated")
	var result struct {
		ID    int64    `json:"id"`
		Names []string `json:"names"`
	}
	response, err := client.invoke(t.Context(), testMethodInfo(t, nil, "demo.Service", "Get"), struct {
		ID int64 `json:"id"`
	}{9007199254740993}, &result, WithTrace(parent))
	if err != nil {
		t.Fatal(err)
	}
	if result.ID != 9007199254740993 || result.Names == nil || response.Header.Get(rpchttp.HeaderPortalTraceID) != parent.ID || response.Server.Name != "demo.server" {
		t.Fatalf("result=%+v metadata=%+v", result, response)
	}
}
func TestRemoteFailuresPreserveStatusAndCause(t *testing.T) {
	for _, test := range []struct {
		name, body, status string
		httpStatus         int
		cause              bool
	}{
		{"application", `{"result":null,"error":{"code":"NOT_FOUND","reason":"missing","message":"not here"}}`, "NOT_FOUND", 404, false},
		{"malformed", `<html>oops</html>`, "SERVICE_UNAVAILABLE", 503, true},
		{"internal status", `{"result":null,"error":{"message":"denied"}}`, "PERMISSION_DENIED", 200, false},
		{"http failure", `{"result":null,"error":null}`, "OK", 502, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reply(w, test.status, test.httpStatus, test.body) }))
			defer server.Close()
			c := newClient(t, Option{
				Identity: testClientIdentity(),
				Endpoint: server.URL,
			})
			response, err := c.invoke(t.Context(), testMethodInfo(t, nil, "demo.Service", "Get"), nil, nil)
			var remote *InvocationError
			if !errors.As(err, &remote) || remote.Metadata != response || response.Status != test.status || (remote.Cause != nil) != test.cause {
				t.Fatalf("response=%+v error=%#v", response, err)
			}
		})
	}
}
func TestMalformedSuccess(t *testing.T) {
	for _, test := range []struct {
		name, body string
		mutate     func(http.Header)
	}{
		{"missing status", `{"result":null}`, func(h http.Header) { h.Del(rpchttp.HeaderRpcStatus) }},
		{"duplicate status", `{"result":null}`, func(h http.Header) { h.Add(rpchttp.HeaderRpcStatus, "OK") }},
		{"invalid server", `{"result":null}`, func(h http.Header) { h.Set(rpchttp.HeaderRpcServer, "bad") }},
		{"wrong content type", `{"result":null}`, func(h http.Header) { h.Set("Content-Type", "text/html") }},
		{"duplicate json", `{"result":1,"result":2}`, nil},
		{"trailing", `{"result":1} {}`, nil},
		{"empty", ``, nil},
		{"null envelope", `null`, nil},
		{"missing result", `{"error":null}`, nil},
		{"contradictory", `{"error":{"message":"bad"},"result":null}`, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", rpchttp.ContentTypeJson)
				w.Header().Set(rpchttp.HeaderRpcStatus, "OK")
				w.Header().Set(rpchttp.HeaderRpcServer, serverIdentity)
				if test.mutate != nil {
					test.mutate(w.Header())
				}
				_, _ = io.WriteString(w, test.body)
			}))
			defer server.Close()
			c := newClient(t, Option{
				Identity: testClientIdentity(),
				Endpoint: server.URL,
			})
			metadata, err := c.invoke(t.Context(), testMethodInfo(t, nil, "demo.Service", "Get"), nil, nil)
			var protocol *ProtocolError
			if !errors.As(err, &protocol) || metadata == nil {
				t.Fatalf("expected protocol error, got %v", err)
			}
		})
	}
}
func TestCancellationAndTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.Copy(io.Discard, r.Body); <-r.Context().Done() }))
	defer server.Close()
	c := newClient(t, Option{
		Identity: testClientIdentity(),
		Endpoint: server.URL,
	})
	_, err := c.invoke(t.Context(), testMethodInfo(t, nil, "demo.Service", "Get"), nil, nil, WithTimeout(20*time.Millisecond))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout: %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = c.invoke(ctx, testMethodInfo(t, nil, "demo.Service", "Get"), nil, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
}
func TestRedirectDoesNotReplayCredentials(t *testing.T) {
	var received atomic.Int32
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { received.Add(1) }))
	defer destination.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, destination.URL, 307) }))
	defer server.Close()
	c := newClient(t, Option{
		Identity: testClientIdentity(),
		Endpoint: server.URL,
		Headers: http.Header{
			"Authorization": []string{"key private"},
		},
	})
	_, err := c.invoke(t.Context(), testMethodInfo(t, nil, "demo.Service", "Get"), nil, nil)
	if err == nil || received.Load() != 0 {
		t.Fatalf("redirect followed: %v", err)
	}
}
func TestGzipResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", rpchttp.ContentTypeJson)
		w.Header().Set(rpchttp.HeaderRpcStatus, "OK")
		w.Header().Set(rpchttp.HeaderRpcServer, serverIdentity)
		w.Header().Set("Content-Encoding", "gzip")
		zw := gzip.NewWriter(w)
		_, _ = fmt.Fprintf(zw, `{"result":"%s","error":null}`, strings.Repeat("x", 128))
		_ = zw.Close()
	}))
	defer server.Close()
	c := newClient(t, Option{
		Identity: testClientIdentity(),
		Endpoint: server.URL,
	})
	var result string
	_, err := c.invoke(t.Context(), testMethodInfo(t, nil, "demo.Service", "Get"), nil, &result)
	if err != nil || result != strings.Repeat("x", 128) {
		t.Fatalf("gzip result=%q err=%v", result, err)
	}

}
func TestConcurrentCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reply(w, "OK", 200, `{"result":42,"error":null}`) }))
	defer server.Close()
	c := newClient(t, Option{
		Identity: testClientIdentity(),
		Endpoint: server.URL,
	})
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			var result int
			_, err := c.invoke(t.Context(), testMethodInfo(t, nil, "demo.Service", "Get"), nil, &result)
			if err != nil || result != 42 {
				t.Errorf("call: %d %v", result, err)
			}
		})
	}
	wg.Wait()
}
func TestInvalidConfigurationAndCall(t *testing.T) {
	for _, endpoint := range []string{"", "ftp://host/invoke", "https://user:pass@host", "https://host?a=1", "https://host#fragment"} {
		if _, err := NewClient(Option{
			Identity: testClientIdentity(),
			Endpoint: endpoint,
		}); err == nil {
			t.Errorf("accepted %q", endpoint)
		}
	}
	c := newClient(t, Option{
		Identity: testClientIdentity(),
		Endpoint: "http://localhost",
	})
	if _, err := c.invoke(t.Context(), MethodInfo{}, nil, nil); err == nil {
		t.Fatal("empty method info accepted")
	}
	if _, err := c.invoke(t.Context(), testMethodInfo(t, nil, "demo.Service", "Get"), nil, nil, WithTimeout(-1)); err == nil {
		t.Error("negative timeout accepted")
	}
}

func TestRegisteredMethodEncoding(t *testing.T) {
	for _, flags := range [][2]bool{{false, false}, {true, false}, {false, true}, {true, true}} {
		t.Run(fmt.Sprint(flags), func(t *testing.T) {
			registry := NewRegistry()
			registry.Register(&ServiceSpec{
				SkelName: "demo.Binary",
				Methods: []MethodSpec{{
					SkelName:                    "call",
					ArgumentsContainsBinaryType: flags[0],
					ResultContainsBinaryType:    flags[1],
				}},
			})
			for _, jsonFallback := range []bool{false, true} {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					expectedType, expectedAccept := rpchttp.RequestContentTypes(flags[0], flags[1])
					if r.Header.Get("Content-Type") != expectedType || r.Header.Get("Accept") != expectedAccept {
						t.Errorf("headers: %v", r.Header)
					}
					body, err := io.ReadAll(r.Body)
					if err != nil {
						t.Error(err)
						return
					}
					var raw []byte
					var params struct {
						Value []byte `json:"value"`
					}
					if flags[0] {
						raw, err = rpchttp.DecodeCBORRequest(body)
						if err == nil {
							err = cbor.Unmarshal(raw, &params)
						}
					} else {
						raw, err = rpchttp.DecodeJSONRequest(body)
						if err == nil {
							err = json.Unmarshal(raw, &params)
						}
					}
					if err != nil || string(params.Value) != "hello" {
						t.Errorf("params: %q, %v", params.Value, err)
					}
					w.Header().Set(rpchttp.HeaderRpcStatus, rpchttp.StatusOK)
					w.Header().Set(rpchttp.HeaderRpcServer, serverIdentity)
					if flags[1] && !jsonFallback {
						raw, err := cbor.Marshal(params)
						if err != nil {
							t.Error(err)
							return
						}
						body, err := rpchttp.EncodeCBORResponse(raw, nil)
						if err != nil {
							t.Error(err)
							return
						}
						w.Header().Set("Content-Type", rpchttp.ContentTypeCbor)
						_, _ = w.Write(body)
					} else {
						reply(w, rpchttp.StatusOK, 200, `{"result":{"value":"aGVsbG8="}}`)
					}
				}))
				var result struct {
					Value []byte `json:"value"`
				}
				client := newClient(t, Option{
					Identity: testClientIdentity(),
					Endpoint: server.URL,
				})
				_, err := client.invoke(t.Context(), testMethodInfo(t, registry, "demo.Binary", "call"), struct {
					Value []byte `json:"value"`
				}{[]byte("hello")}, &result)
				server.Close()
				if err != nil || string(result.Value) != "hello" {
					t.Fatalf("result: %q, %v", result.Value, err)
				}
			}
		})
	}
}

func TestRegistryRejectsUnknownCalls(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		reply(w, rpchttp.StatusOK, 200, `{"result":1}`)
	}))
	defer server.Close()
	registry := NewRegistry()
	registry.Register(&ServiceSpec{
		SkelName: "demo.Checked",
		Methods: []MethodSpec{{
			SkelName: "get",
		}},
	})
	client := newClient(t, Option{
		Identity: testClientIdentity(),
		Endpoint: server.URL,
	})
	for _, call := range [][2]string{{"demo.Unknown", "get"}, {"demo.Checked", "unknown"}} {
		if _, err := client.invoke(t.Context(), testMethodInfo(t, registry, call[0], call[1]), nil, nil); err == nil {
			t.Fatal("invalid call accepted")
		}
	}
	if requests.Load() != 0 {
		t.Fatal("invalid calls reached transport")
	}
	var result int
	if _, err := client.invoke(t.Context(), testMethodInfo(t, registry, "demo.Checked", "get"), nil, &result); err != nil || result != 1 {
		t.Fatalf("registered call: result=%d err=%v", result, err)
	}

}

func TestDefaultRegistryGeneratedRegistration(t *testing.T) {
	// A unique contract allows repeated test runs without mutating global registry state.
	service := "generated.Service" + strings.ReplaceAll(NewTrace().ID, "-", "")
	Register(&ServiceSpec{
		SkelName: service,
		Methods: []MethodSpec{{
			SkelName: "get",
		}},
	})
	if _, ok := GetMethodInfo(service, "get"); !ok {
		t.Fatal("default registration missing")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reply(w, rpchttp.StatusOK, 200, `{"result":null}`) }))
	defer server.Close()
	client := newClient(t, Option{
		Identity: testClientIdentity(),
		Endpoint: server.URL,
	})
	if _, err := client.invoke(t.Context(), testMethodInfo(t, defaultRegistry, service, "get"), nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := client.invoke(t.Context(), testMethodInfo(t, defaultRegistry, service, "typo"), nil, nil); err == nil {
		t.Fatal("unknown registered method accepted")
	}
}

func TestRegisteredMethodRejectsUnadvertisedCBOR(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&ServiceSpec{
		SkelName: "demo.Upload",
		Methods: []MethodSpec{{
			SkelName:                    "put",
			ArgumentsContainsBinaryType: true,
		}},
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", rpchttp.ContentTypeCbor)
		w.Header().Set(rpchttp.HeaderRpcStatus, rpchttp.StatusOK)
		w.Header().Set(rpchttp.HeaderRpcServer, serverIdentity)
		body, _ := rpchttp.EncodeCBORResponse([]byte{0xf6}, nil)
		_, _ = w.Write(body)
	}))
	defer server.Close()
	client := newClient(t, Option{
		Identity: testClientIdentity(),
		Endpoint: server.URL,
	})
	_, err := client.invoke(t.Context(), testMethodInfo(t, registry, "demo.Upload", "put"), nil, nil)
	var protocolErr *ProtocolError
	if !errors.As(err, &protocolErr) {
		t.Fatalf("unadvertised CBOR accepted: %v", err)
	}
}

func TestCustomHeadersPreserveValues(t *testing.T) {
	headers := http.Header{
		"x-custom":          {"first", "second"},
		"Idempotency-Key":   {"key"},
		"X-Idempotency-Key": {"other"},
		"X-Empty":           nil,
		"accept":            {"wrong"},
		"ACCEPT":            {"also-wrong"},
		"content-type":      {"wrong"},
		"vrpc-client":       {"wrong"},
		"vrpc-trace":        {"wrong"},
		"vrpc-options":      {"wrong"},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if values := r.Header.Values("X-Custom"); len(values) != 2 || values[0] != "first" || values[1] != "second" {
			t.Errorf("custom values: %v", values)
		}
		if r.Header.Get("Idempotency-Key") != "key" || r.Header.Get("X-Idempotency-Key") != "other" {
			t.Error("missing idempotency headers")
		}
		if r.Header.Get("Accept") != rpchttp.ContentTypeJson || r.Header.Get("Content-Type") != rpchttp.ContentTypeJson {
			t.Errorf("encoding headers: %v", r.Header)
		}
		if _, err := DecodeIdentity(r.Header.Get(rpchttp.HeaderRpcClient)); err != nil {
			t.Error(err)
		}
		if r.Header.Get(rpchttp.HeaderRpcTrace) == "wrong" || !strings.Contains(r.Header.Get(rpchttp.HeaderRpcOptions), "timeout=") {
			t.Errorf("metadata headers: %v", r.Header)
		}
		reply(w, "OK", 200, `{"result":null}`)
	}))
	defer server.Close()
	client := newClient(t, Option{
		Identity: testClientIdentity(),
		Endpoint: server.URL,
		Headers:  headers,
	})
	headers["x-custom"][0] = "changed"
	for range 2 {
		if _, err := client.invoke(t.Context(), testMethodInfo(t, nil, "demo.Service", "Get"), nil, nil); err != nil {
			t.Fatal(err)
		}
	}
}

func testMethodInfo(t *testing.T, registry *Registry, service, method string) MethodInfo {
	t.Helper()
	if registry == nil {
		registry = NewRegistry()
		registry.Register(&ServiceSpec{
			SkelName: service,
			Methods: []MethodSpec{{
				SkelName: method,
			}},
		})
	}
	info, _ := registry.GetMethodInfo(service, method)
	return info
}

func TestFixedAuthorization(t *testing.T) {
	for _, tt := range []struct {
		name        string
		credentials map[string]string
		want        string
	}{
		{name: "configured", credentials: map[string]string{"key": "token"}, want: "key token"},
		{name: "empty", credentials: map[string]string{}, want: ""},
		{name: "nil", want: "raw value"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("Authorization"); got != tt.want {
					t.Errorf("authorization=%q want=%q", got, tt.want)
				}
				reply(w, "OK", 200, `{"result":null}`)
			}))
			defer server.Close()
			client := newClient(t, Option{
				Identity: testClientIdentity(),
				Endpoint: server.URL,
				Headers: http.Header{
					"authorization": {"raw value"},
				},
				Authorization: tt.credentials,
			})
			if tt.credentials != nil {
				tt.credentials["key"] = "changed"
			}
			info := testMethodInfo(t, nil, "demo.Service", "Get")
			for range 2 {
				if _, _, err := client.Invoke[struct{}](t.Context(), info, nil); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
	if _, err := NewClient(Option{
		Identity: testClientIdentity(),
		Endpoint: "http://localhost",
		Authorization: map[string]string{
			"key": "invalid,value",
		},
	}); err == nil {
		t.Fatal("invalid credentials accepted")
	}
}

func testClientIdentity() Identity {
	return Identity{
		Name:       "demo.client",
		Version:    "1.0.0",
		InstanceID: "123e4567-e89b-12d3-a456-426614174001",
	}
}

func TestClientRequiresIdentity(t *testing.T) {
	for _, field := range []string{"all", "name", "version", "instance"} {
		t.Run(field, func(t *testing.T) {
			identity := testClientIdentity()
			switch field {
			case "all":
				identity = Identity{}
			case "name":
				identity.Name = ""
			case "version":
				identity.Version = ""
			case "instance":
				identity.InstanceID = ""
			}

			client, err := NewClient(Option{
				Endpoint: "http://localhost",
				Identity: identity,
			})
			if err == nil || client != nil {
				t.Fatal("incomplete identity accepted")
			}
		})
	}
}
