package integration_test

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"fmt"
	rpchttp "go.yorun.ai/vrpc/transport/http"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/link/ingressinproc"
	"go.yorun.ai/vine/internal/core/meta"
	rpcspec "go.yorun.ai/vine/internal/core/rpc/spec"
	vinehttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubredis"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/access"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/epmgr"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/rpcgw"
	sitespec "go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
	"go.yorun.ai/vine/util/vcode"
	"go.yorun.ai/vrpc"
)

func TestPortalConformance(t *testing.T) {
	endpoint := "link+inproc://vrpc-go/conformance"
	app := meta.MustNewAppWithRandomId("demo.server", "1.0.0")
	ingressinproc.Register(endpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := vinehttp.CheckRequestHeaders(r.Header); err != nil {
			t.Error(err)
		}
		if _, err := vinehttp.DecodeTraceFromHeader(r.Header); err != nil {
			t.Error(err)
		}
		if _, err := vinehttp.DecodeClientFromHeader(r.Header); err != nil {
			t.Error(err)
		}
		if _, err := vinehttp.DecodeInitiatorFromHeader(r.Header); err != nil {
			t.Error(err)
		}
		if r.Header.Get(vinehttp.HeaderRpcActor) == "" {
			t.Error("Portal did not set an actor")
		}
		write := func(result any, err ex.Error) {
			if e := vinehttp.WriteResponse(w, r, &rpcspec.ResponseImpl{
				ServerValue: app,
				ResultValue: result,
				ErrorValue:  err,
			}); e != nil {
				t.Error(e)
			}
		}
		if strings.HasSuffix(r.URL.Path, "/demo.Auth/Auth") {
			var request struct {
				Params struct {
					Credential map[string]string `json:"credential"`
				} `json:"params"`
			}
			body, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(body, &request); err != nil {
				t.Error(err)
			}
			if request.Params.Credential["key"] != "secret" {
				write(nil, ex.New(ex.Unauthorized, "invalid key"))
				return
			}
			write(map[string]string{
				"id": "user",
			}, ex.NewOK())
			return
		}
		if strings.HasSuffix(r.URL.Path, "/Fail") {
			write(nil, ex.New(ex.NotFound, "missing", ex.WithReason("gone")))
			return
		}
		body, err := vinehttp.ReadRequestBody(r)
		if err != nil {
			t.Error(err)
		}
		var request struct {
			Params struct {
				Data []byte `json:"data"`
				ID   int64  `json:"id"`
			} `json:"params"`
		}
		if r.Header.Get("Content-Type") == rpchttp.ContentTypeCbor {
			if err := cbor.Unmarshal(body, &request); err != nil {
				t.Error(err)
			}
			encoded := vcode.MustMarshalCbor(map[string]any{
				"result": request.Params,
				"error":  nil,
			})
			w.Header().Set("Content-Type", rpchttp.ContentTypeCbor)
			vinehttp.EncodeServerToHeader(w.Header(), app)
			vinehttp.EncodeStatusCodeToHeader(w.Header(), ex.OK)
			_, _ = w.Write(encoded)
		} else {
			if err := json.Unmarshal(body, &request); err != nil {
				t.Error(err)
			}
			write(request.Params, ex.NewOK())
		}
	}))
	t.Cleanup(func() { ingressinproc.Unregister(endpoint) })
	values := map[string]string{
		redised.FormatSchemaActorKey("demo.UserActor"): vcode.MustMarshalJsonS(redised.SchemaActor{
			SkelName: "demo.UserActor", AuthEnabled: true, AuthCredential: &skel.DataSchema{
				Members: []*skel.MemberSchema{{
					Name: "key",
				}},
			}, AuthInfo: &skel.DataSchema{
				SkelName: "demo.UserInfo",
			}, IdentifierField: "id",
			AuthService: &skel.ServiceSchema{
				SkelName: "demo.Auth",
			}, AuthMethod: &skel.MethodSchema{
				SkelName: "Auth",
			},
		}),
		redised.FormatSchemaServiceKey("demo.Service"): vcode.MustMarshalJsonS(redised.SchemaService{
			SkelName: "demo.Service", AuthMode: skel.AuthModeNoAuth, Audiences: []*skel.ActorAudienceSchema{{
				SkelName: "demo.UserActor",
			}},
			Methods: []*skel.MethodSchema{{
				SkelName: "Get",
			}, {
				SkelName: "Secure",
				AuthMode: skel.AuthModeAuth,
			}, {
				SkelName: "Fail",
			}},
		}),
	}
	for _, service := range []string{"demo.Service", "demo.Auth"} {
		values[redised.FormatRpcServiceRegistrationKey(service, "demo.server", "instance")] = vcode.MustMarshalJsonS(redised.RpcServiceRegistration{
			Endpoint:      endpoint,
			ServiceName:   service,
			AppName:       "demo.server",
			AppInstanceId: "instance",
		})
	}
	redis := hubredis.NewTestClient(values)
	manager := &epmgr.Manager{
		Context: t.Context(),
		Redis:   redis,
	}
	manager.DIInit()
	admission := &access.Access{
		Context: t.Context(),
		Redis:   redis,
		Epmgr:   manager,
	}
	admission.DIInit()
	gateway := rpcgw.New(t.Context(), app, admission, manager, redised.PortalSite{
		Name: "test",
		Type: "RPCGW",
		ActorVia: redised.PortalActorVia{
			ActorSkelName: "demo.UserActor",
		},
		RpcgwConfig: &redised.PortalRpcgwConfig{
			Services: []redised.PortalRpcgwService{{
				SkelName: "demo.Service",
			}},
		},
	})
	t.Cleanup(gateway.Stop)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gateway.Serve(&sitespec.Context{
			Request:        r,
			ResponseWriter: w,
			RemoteAddr:     "127.0.0.1",
		})
	}))
	t.Cleanup(server.Close)
	params := struct {
		Data []byte `json:"data"`
		ID   int64  `json:"id"`
	}{[]byte{0, 255}, 9007199254740993}
	for _, binary := range []bool{false, true} {
		t.Run(fmt.Sprint("binary=", binary), func(t *testing.T) {
			registry := vrpc.NewRegistry()
			registry.Register(&vrpc.ServiceSpec{
				SkelName: "demo.Service",
				Methods: []vrpc.MethodSpec{{
					SkelName:                    "Get",
					ArgumentsContainsBinaryType: binary,
					ResultContainsBinaryType:    binary,
				}, {
					SkelName:                    "Secure",
					ArgumentsContainsBinaryType: binary,
					ResultContainsBinaryType:    binary,
				}, {
					SkelName:                 "Fail",
					ResultContainsBinaryType: binary,
				}},
			})
			client, err := vrpc.NewClient(vrpc.Option{
				Identity: testClientIdentity(),
				Endpoint: server.URL + "/invoke",
				Authorization: map[string]string{
					"key": "secret",
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			for _, method := range []string{"Get", "Secure"} {
				type resultType struct {
					Data []byte `json:"data"`
					ID   int64  `json:"id"`
				}
				result, response, err := client.InvokeAs[resultType](t.Context(), registeredMethod(t, registry, method), params)
				if err != nil || result.ID != params.ID || !bytes.Equal(result.Data, params.Data) {
					t.Fatalf("%s result=%+v response=%+v err=%v", method, result, response, err)
				}
				if response.Header.Get(rpchttp.HeaderPortalTraceID) == "" {
					t.Error("missing Portal trace")
				}
			}
			_, response, err := client.InvokeAs[struct{}](t.Context(), registeredMethod(t, registry, "Fail"), nil)
			var remote *vrpc.InvocationError
			if !errors.As(err, &remote) || response.Status != "NOT_FOUND" || response.HTTPStatus != 404 || remote.Payload.Reason != "gone" {
				t.Fatalf("business error: %+v %v", response, err)
			}
			_, response, err = client.InvokeAs[struct{}](t.Context(), registeredMethod(t, registry, "Get"), nil, vrpc.WithTimeout(121*time.Second))
			if !errors.As(err, &remote) || response.Status != "INVALID_REQUEST" {
				t.Fatalf("gateway timeout limit: %+v %v", response, err)
			}
		})
	}
	t.Run("raw client", func(t *testing.T) {
		for _, credential := range []string{"secret", "", "incorrect"} {
			t.Run("credential="+credential, func(t *testing.T) {
				client, err := vrpc.NewClient(vrpc.Option{
					Identity: testClientIdentity(),
					Endpoint: server.URL + "/invoke",
					Headers: http.Header{
						"Authorization": []string{"key " + credential},
					},
				})
				if err != nil {
					t.Fatal(err)
				}
				result, metadata, err := client.InvokeRaw(t.Context(), "demo.Service", "Secure", map[string]any{
					"id":   42,
					"data": "AP8=",
				})
				if credential != "secret" {
					var remote *vrpc.InvocationError
					if result != nil || !errors.As(err, &remote) || metadata == nil || metadata.Status != "UNAUTHORIZED" || remote.Metadata != metadata {
						t.Fatalf("raw authentication: result=%v metadata=%+v err=%v", result, metadata, err)
					}
					return
				}
				if err != nil || metadata == nil || metadata.Status != "OK" || metadata.Header.Get(rpchttp.HeaderPortalTraceID) == "" {
					t.Fatalf("raw invocation: metadata=%+v err=%v", metadata, err)
				}
				object, ok := result.(map[string]any)
				if !ok || object["id"] != float64(42) || object["data"] != "AP8=" {
					t.Fatalf("raw result: %#v", result)
				}

				result, metadata, err = client.InvokeRaw(t.Context(), "demo.Service", "Fail", nil)
				var remote *vrpc.InvocationError
				if result != nil || !errors.As(err, &remote) || metadata == nil || metadata.Status != "NOT_FOUND" || metadata.HTTPStatus != 404 || remote.Metadata != metadata || remote.Payload == nil || remote.Payload.Reason != "gone" {
					t.Fatalf("raw business error: result=%v metadata=%+v err=%v", result, metadata, err)
				}
			})
		}
	})
	t.Run("registered client", func(t *testing.T) {
		registry := vrpc.NewRegistry()
		registry.Register(&vrpc.ServiceSpec{SkelName: "demo.Service", Methods: []vrpc.MethodSpec{
			{
				SkelName:                    "Get",
				ArgumentsContainsBinaryType: true,
				ResultContainsBinaryType:    true,
			},
			{
				SkelName:                    "Secure",
				ArgumentsContainsBinaryType: true,
				ResultContainsBinaryType:    true,
			},
			{
				SkelName:                 "Fail",
				ResultContainsBinaryType: true,
			},
		}})
		client, err := vrpc.NewClient(vrpc.Option{
			Identity: testClientIdentity(),
			Endpoint: server.URL + "/invoke",
			Headers: http.Header{
				"Authorization": []string{"key secret"},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		for _, method := range []string{"Get", "Secure"} {
			type resultType struct {
				Data []byte `json:"data"`
				ID   int64  `json:"id"`
			}
			result, response, err := client.InvokeAs[resultType](t.Context(), registeredMethod(t, registry, method), params)
			if err != nil || result.ID != params.ID || !bytes.Equal(result.Data, params.Data) {
				t.Fatalf("registered %s: %+v, %+v, %v", method, result, response, err)
			}
		}
		_, response, err := client.InvokeAs[struct{}](t.Context(), registeredMethod(t, registry, "Fail"), nil)
		var remote *vrpc.InvocationError
		if !errors.As(err, &remote) || response.Status != "NOT_FOUND" || remote.Payload.Reason != "gone" {
			t.Fatalf("registered error: %+v, %v", response, err)
		}
	})
	registry := vrpc.NewRegistry()
	registry.Register(&vrpc.ServiceSpec{
		SkelName: "demo.Service",
		Methods: []vrpc.MethodSpec{{
			SkelName: "Secure",
		}},
	})
	for _, credential := range []string{"", "key incorrect"} {
		client, _ := vrpc.NewClient(vrpc.Option{
			Identity: testClientIdentity(),
			Endpoint: server.URL + "/invoke",
			Headers: http.Header{
				"Authorization": []string{credential},
			},
		})
		_, response, err := client.InvokeAs[struct{}](t.Context(), registeredMethod(t, registry, "Secure"), params)
		var remote *vrpc.InvocationError
		if !errors.As(err, &remote) || response.Status != "UNAUTHORIZED" {
			t.Fatalf("credential rejection: %+v %v", response, err)
		}
	}
}

func registeredMethod(t *testing.T, registry *vrpc.Registry, name string) vrpc.MethodInfo {
	t.Helper()
	info, ok := registry.GetMethodInfo("demo.Service", name)
	if !ok {
		t.Fatalf("missing method %s", name)
	}
	return info
}

func testClientIdentity() vrpc.Identity {
	return vrpc.Identity{
		Name:       "demo.client",
		Version:    "1.0.0",
		InstanceID: "123e4567-e89b-12d3-a456-426614174001",
	}
}
