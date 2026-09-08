package integration_test

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	wire "github.com/fxamacker/cbor/v2"
	"go.yorun.ai/vine/internal/core/ex"
	"go.yorun.ai/vine/internal/core/link/ingressinproc"
	"go.yorun.ai/vine/internal/core/meta"
	rpcspec "go.yorun.ai/vine/internal/core/rpc/spec"
	rpchttp "go.yorun.ai/vine/internal/core/rpc/transport/http"
	"go.yorun.ai/vine/internal/core/skel"
	"go.yorun.ai/vine/internal/daemon/hub/api/redised"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/comp/hubredis"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/access"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/epmgr"
	"go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/rpcgw"
	sitespec "go.yorun.ai/vine/internal/daemon/portal/src/server/mod/site/spec"
	"go.yorun.ai/vine/util/vcode"
	"go.yorun.ai/vrpc"
	"go.yorun.ai/vrpc/codec/cbor"
)

func TestPortalConformance(t *testing.T) {
	endpoint := "link+inproc://vrpc-go/conformance"
	app := meta.MustNewAppWithRandomId("demo.server", "1.0.0")
	ingressinproc.Register(endpoint, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := rpchttp.CheckRequestHeaders(r.Header); err != nil {
			t.Error(err)
		}
		if _, err := rpchttp.DecodeTraceFromHeader(r.Header); err != nil {
			t.Error(err)
		}
		if _, err := rpchttp.DecodeClientFromHeader(r.Header); err != nil {
			t.Error(err)
		}
		if _, err := rpchttp.DecodeInitiatorFromHeader(r.Header); err != nil {
			t.Error(err)
		}
		if r.Header.Get(rpchttp.HeaderRpcActor) == "" {
			t.Error("Portal did not set an actor")
		}
		write := func(result any, err ex.Error) {
			if e := rpchttp.WriteResponse(w, r, &rpcspec.ResponseImpl{ServerValue: app, ResultValue: result, ErrorValue: err}); e != nil {
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
			write(map[string]string{"id": "user"}, ex.NewOK())
			return
		}
		if strings.HasSuffix(r.URL.Path, "/Fail") {
			write(nil, ex.New(ex.NotFound, "missing", ex.WithReason("gone")))
			return
		}
		body, err := rpchttp.ReadRequestBody(r)
		if err != nil {
			t.Error(err)
		}
		var request struct {
			Params struct {
				Data []byte `json:"data"`
				ID   int64  `json:"id"`
			} `json:"params"`
		}
		if r.Header.Get("Content-Type") == vrpc.ContentTypeCBOR {
			if err := wire.Unmarshal(body, &request); err != nil {
				t.Error(err)
			}
			encoded := vcode.MustMarshalCbor(map[string]any{"result": request.Params, "error": nil})
			w.Header().Set("Content-Type", vrpc.ContentTypeCBOR)
			rpchttp.EncodeServerToHeader(w.Header(), app)
			rpchttp.EncodeStatusCodeToHeader(w.Header(), ex.OK)
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
			SkelName: "demo.UserActor", AuthEnabled: true, AuthCredential: &skel.DataSchema{Members: []*skel.MemberSchema{{Name: "key"}}}, AuthInfo: &skel.DataSchema{SkelName: "demo.UserInfo"}, IdentifierField: "id",
			AuthService: &skel.ServiceSchema{SkelName: "demo.Auth"}, AuthMethod: &skel.MethodSchema{SkelName: "Auth"},
		}),
		redised.FormatSchemaServiceKey("demo.Service"): vcode.MustMarshalJsonS(redised.SchemaService{
			SkelName: "demo.Service", AuthMode: skel.AuthModeNoAuth, Audiences: []*skel.ActorAudienceSchema{{SkelName: "demo.UserActor"}},
			Methods: []*skel.MethodSchema{{SkelName: "Get"}, {SkelName: "Secure", AuthMode: skel.AuthModeAuth}, {SkelName: "Fail"}},
		}),
	}
	for _, service := range []string{"demo.Service", "demo.Auth"} {
		values[redised.FormatRpcServiceRegistrationKey(service, "demo.server", "instance")] = vcode.MustMarshalJsonS(redised.RpcServiceRegistration{Endpoint: endpoint, ServiceName: service, AppName: "demo.server", AppInstanceId: "instance"})
	}
	redis := hubredis.NewTestClient(values)
	manager := &epmgr.Manager{Context: t.Context(), Redis: redis}
	manager.DIInit()
	admission := &access.Access{Context: t.Context(), Redis: redis, Epmgr: manager}
	admission.DIInit()
	gateway := rpcgw.New(t.Context(), app, admission, manager, redised.PortalSite{Name: "test", Type: "RPCGW", ActorVia: redised.PortalActorVia{ActorSkelName: "demo.UserActor"}, RpcgwConfig: &redised.PortalRpcgwConfig{Services: []redised.PortalRpcgwService{{SkelName: "demo.Service"}}}})
	t.Cleanup(gateway.Stop)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gateway.Serve(&sitespec.Context{Request: r, ResponseWriter: w, RemoteAddr: "127.0.0.1"})
	}))
	t.Cleanup(server.Close)
	params := struct {
		Data []byte `json:"data"`
		ID   int64  `json:"id"`
	}{[]byte{0, 255}, 9007199254740993}
	for _, codec := range []vrpc.Codec{vrpc.JSONCodec{}, cbor.Codec{}} {
		t.Run(codec.ContentType(), func(t *testing.T) {
			client, err := vrpc.NewClient(vrpc.Options{Endpoint: server.URL + "/invoke", Codec: codec, Headers: http.Header{"Authorization": []string{"key secret"}}})
			if err != nil {
				t.Fatal(err)
			}
			for _, method := range []string{"Get", "Secure"} {
				var result struct {
					Data []byte `json:"data"`
					ID   int64  `json:"id"`
				}
				response, err := client.Call(t.Context(), "demo.Service", method, params, &result)
				if err != nil || result.ID != params.ID || !bytes.Equal(result.Data, params.Data) {
					t.Fatalf("%s result=%+v response=%+v err=%v", method, result, response, err)
				}
				if response.PortalTraceID == "" {
					t.Error("missing Portal trace")
				}
			}
			response, err := client.Call(t.Context(), "demo.Service", "Fail", nil, nil)
			var remote *vrpc.InvocationError
			if !errors.As(err, &remote) || response.Status != "NOT_FOUND" || response.HTTPStatus != 404 || remote.Payload.Reason != "gone" {
				t.Fatalf("business error: %+v %v", response, err)
			}
			response, err = client.Call(t.Context(), "demo.Service", "Get", nil, nil, vrpc.WithTimeout(121*time.Second))
			if !errors.As(err, &remote) || response.Status != "INVALID_REQUEST" {
				t.Fatalf("gateway timeout limit: %+v %v", response, err)
			}
		})
	}
	for _, credential := range []string{"", "key incorrect"} {
		client, _ := vrpc.NewClient(vrpc.Options{Endpoint: server.URL + "/invoke", Headers: http.Header{"Authorization": []string{credential}}})
		response, err := client.Call(t.Context(), "demo.Service", "Secure", params, nil)
		var remote *vrpc.InvocationError
		if !errors.As(err, &remote) || response.Status != "UNAUTHORIZED" {
			t.Fatalf("credential rejection: %+v %v", response, err)
		}
	}
}
