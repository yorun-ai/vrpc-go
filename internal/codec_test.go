package vrpc

import (
	"bytes"
	"context"
	"github.com/fxamacker/cbor/v2"
	rpchttp "go.yorun.ai/vrpc/transport/http"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBinaryAndJSONFallback(t *testing.T) {
	for _, fallback := range []bool{false, true} {
		t.Run(map[bool]string{
			false: "binary",
			true:  "json fallback",
		}[fallback], func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Content-Type") != rpchttp.ContentTypeCbor {
					t.Error("expected cbor request")
				}
				var request struct {
					Params struct {
						Data  []byte           `cbor:"data"`
						Items []string         `cbor:"items"`
						Map   map[int64][]byte `cbor:"map"`
					} `cbor:"params"`
				}
				body, _ := io.ReadAll(r.Body)
				if err := cbor.Unmarshal(body, &request); err != nil {
					t.Error(err)
				}
				if !bytes.Equal(request.Params.Data, []byte{0, 255}) || request.Params.Items == nil || !bytes.Equal(request.Params.Map[9007199254740993], []byte{42}) {
					t.Errorf("request=%+v", request)
				}
				w.Header().Set(rpchttp.HeaderRpcStatus, "OK")
				w.Header().Set(rpchttp.HeaderRpcServer, "name=demo.server,version=1.0.0,instanceId=123e4567-e89b-12d3-a456-426614174000")
				if fallback {
					w.Header().Set("Content-Type", rpchttp.ContentTypeJson)
					_, _ = io.WriteString(w, `{"result":{"data":"AP8="},"error":null}`)
					return
				}
				w.Header().Set("Content-Type", rpchttp.ContentTypeCbor)
				encoded, _ := cbor.Marshal(map[string]any{
					"result": map[string]any{
						"data": []byte{0, 255},
					},
					"error": nil,
				})
				_, _ = w.Write(encoded)
			}))
			defer server.Close()
			registry := NewRegistry()
			registry.Register(&ServiceSpec{
				SkelName: "demo.Service",
				Methods: []MethodSpec{{
					SkelName:                    "Get",
					ArgumentsContainsBinaryType: true,
					ResultContainsBinaryType:    true,
				}},
			})
			client, err := NewClient(Option{
				Identity: testClientIdentity(),
				Endpoint: server.URL,
			})
			if err != nil {
				t.Fatal(err)
			}
			params := struct {
				Data  []byte           `json:"data" cbor:"data"`
				Items []string         `json:"items" cbor:"items"`
				Map   map[int64][]byte `json:"map" cbor:"map"`
			}{
				Data: []byte{0, 255},
				Map: map[int64][]byte{
					9007199254740993: {42},
				},
			}
			var result struct {
				Data []byte `json:"data" cbor:"data"`
			}
			_, err = client.invoke(context.Background(), testMethodInfo(t, registry, "demo.Service", "Get"), params, &result)
			if err != nil || !bytes.Equal(result.Data, params.Data) {
				t.Fatalf("result=%+v err=%v", result, err)
			}
		})
	}
}
func TestMalformedCBOR(t *testing.T) {
	codec := _Codec(true)
	for _, data := range [][]byte{nil, {0xf6}, {0xa0}, {0xa2, 0x66, 'r', 'e', 's', 'u', 'l', 't', 0x01, 0x66, 'r', 'e', 's', 'u', 'l', 't', 0x02}} {
		if _, err := codec.DecodeResponse(data, nil); err == nil {
			t.Errorf("accepted %x", data)
		}
	}
	data, _ := cbor.Marshal(map[string]any{
		"result": nil,
		"error": map[string]string{
			"code":    "NOT_FOUND",
			"message": "missing",
		},
	})
	payload, err := codec.DecodeResponse(data, nil)
	if err != nil || payload.Code != "NOT_FOUND" {
		t.Fatalf("payload=%+v %v", payload, err)
	}
}
