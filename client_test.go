package vrpc_test

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

	"go.yorun.ai/vrpc"
)

const serverIdentity = "name=demo.server,version=1.0.0,instanceId=123e4567-e89b-12d3-a456-426614174000"

func reply(w http.ResponseWriter, status string, httpStatus int, body string) {
	w.Header().Set("Content-Type", vrpc.ContentTypeJSON)
	w.Header().Set(vrpc.HeaderStatus, status)
	w.Header().Set(vrpc.HeaderServer, serverIdentity)
	w.WriteHeader(httpStatus)
	_, _ = io.WriteString(w, body)
}
func newClient(t *testing.T, options vrpc.Options) *vrpc.Client {
	t.Helper()
	c, err := vrpc.NewClient(options)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func TestCallWireAndIdentity(t *testing.T) {
	parent := vrpc.NewTrace()
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
		if _, err := vrpc.DecodeIdentity(r.Header.Get(vrpc.HeaderClient)); err != nil {
			t.Error(err)
		}
		if !strings.Contains(r.Header.Get(vrpc.HeaderTrace), parent.ID) || strings.Contains(r.Header.Get(vrpc.HeaderTrace), parent.Span) {
			t.Error("trace not derived")
		}
		if !strings.HasPrefix(r.Header.Get(vrpc.HeaderOptions), "timeout=") {
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
		w.Header().Set(vrpc.HeaderPortalTraceID, parent.ID)
		reply(w, "OK", 200, `{"result":{"id":9007199254740993,"names":[]},"error":null}`)
	}))
	defer server.Close()
	headers := http.Header{"X-Example": []string{"original"}}
	client := newClient(t, vrpc.Options{Endpoint: server.URL + "/prefix/invoke/", Headers: headers, Authorization: func(ctx context.Context) (string, error) {
		return vrpc.EncodeCredentials(map[string]string{"key": "token"})
	}})
	headers.Set("X-Example", "mutated")
	var result struct {
		ID    int64    `json:"id"`
		Names []string `json:"names"`
	}
	response, err := client.Call(t.Context(), "demo.Service", "Get", struct {
		ID int64 `json:"id"`
	}{9007199254740993}, &result, vrpc.WithTrace(parent))
	if err != nil {
		t.Fatal(err)
	}
	if result.ID != 9007199254740993 || result.Names == nil || response.PortalTraceID != parent.ID || response.Server.Name != "demo.server" {
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
			c := newClient(t, vrpc.Options{Endpoint: server.URL})
			response, err := c.Call(t.Context(), "demo.Service", "Get", nil, nil)
			var remote *vrpc.InvocationError
			if !errors.As(err, &remote) || remote.Response != response || response.Status != test.status || (remote.Cause != nil) != test.cause {
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
		{"missing status", `{"result":null}`, func(h http.Header) { h.Del(vrpc.HeaderStatus) }},
		{"duplicate status", `{"result":null}`, func(h http.Header) { h.Add(vrpc.HeaderStatus, "OK") }},
		{"invalid server", `{"result":null}`, func(h http.Header) { h.Set(vrpc.HeaderServer, "bad") }},
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
				w.Header().Set("Content-Type", vrpc.ContentTypeJSON)
				w.Header().Set(vrpc.HeaderStatus, "OK")
				w.Header().Set(vrpc.HeaderServer, serverIdentity)
				if test.mutate != nil {
					test.mutate(w.Header())
				}
				_, _ = io.WriteString(w, test.body)
			}))
			defer server.Close()
			c := newClient(t, vrpc.Options{Endpoint: server.URL})
			metadata, err := c.Call(t.Context(), "demo.Service", "Get", nil, nil)
			var protocol *vrpc.ProtocolError
			if !errors.As(err, &protocol) || metadata == nil {
				t.Fatalf("expected protocol error, got %v", err)
			}
		})
	}
}
func TestCancellationAndTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.Copy(io.Discard, r.Body); <-r.Context().Done() }))
	defer server.Close()
	c := newClient(t, vrpc.Options{Endpoint: server.URL})
	_, err := c.Call(t.Context(), "demo.Service", "Get", nil, nil, vrpc.WithTimeout(20*time.Millisecond))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout: %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = c.Call(ctx, "demo.Service", "Get", nil, nil)
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
	c := newClient(t, vrpc.Options{Endpoint: server.URL, Headers: http.Header{"Authorization": []string{"key private"}}})
	_, err := c.Call(t.Context(), "demo.Service", "Get", nil, nil)
	if err == nil || received.Load() != 0 {
		t.Fatalf("redirect followed: %v", err)
	}
}
func TestGzipAndDecodedBodyLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", vrpc.ContentTypeJSON)
		w.Header().Set(vrpc.HeaderStatus, "OK")
		w.Header().Set(vrpc.HeaderServer, serverIdentity)
		w.Header().Set("Content-Encoding", "gzip")
		zw := gzip.NewWriter(w)
		_, _ = fmt.Fprintf(zw, `{"result":"%s","error":null}`, strings.Repeat("x", 128))
		_ = zw.Close()
	}))
	defer server.Close()
	for _, limit := range []int64{64, 1024} {
		c := newClient(t, vrpc.Options{Endpoint: server.URL, MaxResponseBytes: limit})
		var result string
		_, err := c.Call(t.Context(), "demo.Service", "Get", nil, &result)
		if (err != nil) != (limit == 64) {
			t.Fatalf("limit %d: %v", limit, err)
		}
	}
}
func TestConcurrentCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reply(w, "OK", 200, `{"result":42,"error":null}`) }))
	defer server.Close()
	c := newClient(t, vrpc.Options{Endpoint: server.URL})
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			var result int
			_, err := c.Call(t.Context(), "demo.Service", "Get", nil, &result)
			if err != nil || result != 42 {
				t.Errorf("call: %d %v", result, err)
			}
		})
	}
	wg.Wait()
}
func TestInvalidConfigurationAndCall(t *testing.T) {
	for _, endpoint := range []string{"", "ftp://host/invoke", "https://user:pass@host", "https://host?a=1", "https://host#fragment"} {
		if _, err := vrpc.NewClient(vrpc.Options{Endpoint: endpoint}); err == nil {
			t.Errorf("accepted %q", endpoint)
		}
	}
	for _, header := range []string{"VrPc-AcToR", "Accept", "Content-Type", "Idempotency-Key"} {
		if _, err := vrpc.NewClient(vrpc.Options{Endpoint: "http://localhost", Headers: http.Header{header: []string{"x"}}}); err == nil {
			t.Errorf("accepted header %s", header)
		}
	}
	c := newClient(t, vrpc.Options{Endpoint: "http://localhost"})
	for _, service := range []string{"../bad", "..", "a?b", "a#b"} {
		if _, err := c.Call(t.Context(), service, "Get", nil, nil); err == nil {
			t.Errorf("accepted service %s", service)
		}
	}
	if _, err := c.Call(t.Context(), "demo.Service", "Get", nil, nil, vrpc.WithTimeout(-1)); err == nil {
		t.Error("negative timeout accepted")
	}
}
