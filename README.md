# vrpc-go

A standalone Go client for [vRPC](https://github.com/yorun-ai/vrpc-ts), compatible with [Vine Portal](https://github.com/yorun-ai/vine).

**English** | [简体中文](README.zh-CN.md)

- Module: `go.yorun.ai/vrpc`; package: `vrpc`.
- Go 1.27 or later. The JSON client uses only the Go standard library.
- Optional CBOR/Binary support: `go.yorun.ai/vrpc/codec/cbor` (fxamacker/cbor).
- No dependency on Vine, application lifecycle, DI, actor registry, or framework logger.
- Apache License 2.0. Initial implementation; the public API is not yet versioned.

## Try the client

Clone [yorun-ai/vrpc-go](https://github.com/yorun-ai/vrpc-go) and run `go test ./...`.
The intended import path is `go.yorun.ai/vrpc`. Until its vanity import mapping and a release are published, use a local checkout from a consuming module:

```sh
go mod edit -require=go.yorun.ai/vrpc@v0.0.0
go mod edit -replace=go.yorun.ai/vrpc=/absolute/path/to/vrpc-go
go mod tidy
```

```go
client, err := vrpc.NewClient(vrpc.Options{
    Endpoint: "https://api.example.com/invoke",
    Identity: vrpc.Identity{Name: "demo.client", Version: "1.0.0"},
    Authorization: func(ctx context.Context) (string, error) {
        return vrpc.EncodeCredentials(map[string]string{"key": token})
    },
})
if err != nil {
    return err
}

var result struct {
    Name string `json:"name"`
}
metadata, err := client.Call(ctx, "demo.UserService", "Get",
    struct { ID int64 `json:"id"` }{ID: 42}, &result)
```

Import `context` and `go.yorun.ai/vrpc`. The example uses the caller's `ctx` and `token`.
Use your service's exact contract names. `Endpoint` includes the invocation prefix;
the client appends `/<service>/<method>` and does not automatically add `/invoke`.

A complete command-line example is included:

```sh
# Set PORTAL_AUTHORIZATION in your environment for authenticated calls.
go run ./examples/portal -endpoint https://api.example.com/invoke \
  -service demo.UserService -method Get -params '{"id":42}'
```

## Configuration and behavior

`Identity` defaults to `vrpc.client`, version `0.0.0`, and a newly generated UUID.
Use lowercase letters separated by dots for the name and a full semantic version.
The instance UUID is stable for that client. Each call creates a trace and span;
`vrpc.WithTrace(parent)` preserves a parent trace ID and creates a child span.

`Options.Timeout` defaults to 30 seconds. `vrpc.WithTimeout(duration)` overrides
it for a call; an earlier context deadline takes precedence. The remaining
duration is sent in `vrpc-options`. Current Portal defaults to 30 seconds and
rejects values above 120 seconds. The client does not hardcode that gateway limit.

Client cancellation stops waiting, but Portal may continue execution until its
own deadline. The client does not add invocation retries or follow redirects.
Custom transports are responsible for their own behavior. No idempotency header
is added. Framework Actor and Initiator values are not synthesized by the client.

`Authorization` runs once per invocation with the call context, allowing credential
refresh. An empty value sends no authorization header. `EncodeCredentials` formats
Portal fields as `field value, field value` in sorted order; it rejects commas,
control characters, empty values, and case-insensitive duplicate field names.
It does not assume Bearer authentication or authenticate locally.

`Headers` accepts additional single-valued HTTP headers and is copied at construction.
vRPC, encoding, framing, and idempotency headers are reserved. `Transport` accepts
an `http.RoundTripper` for custom TLS or networking; the caller owns its lifecycle.
Clients support concurrent calls when custom codecs, transports, and credential
callbacks do too.

## JSON and Binary

JSON is the default. Typed structs with `json` tags preserve Go integer precision.
An `any`/map result follows standard JSON decoding rules; use typed fields when
integer precision matters. Nil slices/maps encode as empty collections under
`encoding/json/v2`, matching current Vine encoding. No schema registration is needed.

For Binary arguments/results, configure the optional codec:

```go
import "go.yorun.ai/vrpc/codec/cbor"

client, err := vrpc.NewClient(vrpc.Options{
    Endpoint: endpoint,
    Codec: cbor.Codec{},
})
```

This sends CBOR requests and accepts CBOR or JSON responses (Portal can return
JSON errors even for CBOR requests). `[]byte` uses CBOR byte strings, integer map
keys remain integers, and nil collections encode as empty collections. Use
matching `json` and `cbor` field names; fxamacker also falls back to `json` tags.
The JSON-only client rejects unexpected CBOR responses. Compression is handled
by the transport: the default Go transport negotiates and decodes gzip. No zstd
support is advertised by this package.

## Errors and metadata

A call succeeds only with HTTP 2xx and `vrpc-status: OK`. On success, the response
must have valid `content-type`, `vrpc-status`, `vrpc-server`, and an envelope with
a result and no error. Unknown uppercase status codes remain observable for
forward compatibility.

- `*vrpc.InvocationError` preserves a failing remote status or non-2xx HTTP response,
  an optional `ErrorPayload`, and any decoding failure in `Cause`.
- `*vrpc.ProtocolError` identifies malformed successful responses.
- Transport/authorization/context errors wrap their causes; use `errors.Is` for
  cancellation/deadline errors and `errors.As` for structured errors.

`Response` exposes HTTP status, protocol status, server identity, request trace,
response headers, and `portal-trace-id`. Classify remote outcomes by `Response.Status`,
not message text or the auxiliary payload's `Code`. `MaxResponseBytes` defaults
to 16 MiB and limits the body after transport decompression. No response body is
included in protocol error messages.

## Shared transport

`transport/http` is extracted from Vine's existing HTTP transport. Both the standalone
client and the Vine framework adapter use its wire constants, header checks,
timeout handling, raw JSON/CBOR envelopes, body limits, and HTTP exchange lifecycle.
`transport/http/cbor` remains optional for JSON-only callers.

Vine retains `meta`, `MethodInfo`, per-schema encoding and validation, framework
error mapping, and h2c/mTLS configuration. Encoded arguments and results pass through
the shared envelopes unchanged, including legacy null-collection profiles. See
[the transport boundary](transport/http/README.md) and [NOTICE](NOTICE) for provenance.

## Compatibility and scope

Tests exercise real Portal RpcGW and authentication code from Vine v0.15.3 over
HTTP, with an in-process backend fixture and in-memory discovery/schema fixtures.
They cover JSON/CBOR, authenticated and anonymous methods, authentication rejection,
business errors, trace propagation, and the gateway timeout limit. They do not
simulate a complete deployed Hub/Link cluster or backend mTLS.

This first version supports hand-written request/result types. Existing skelc Go
service clients still depend on Vine and cannot be plugged into this client
unchanged. Generator adaptation remains follow-up work; Vine consumes the shared transport
through its framework adapter, rather than the standalone client API. The client does not implement Portal `/inspect`.

## Development

```sh
GOWORK=off go test -race ./...
GOWORK=off go vet ./...
cd integration
GOWORK=off go test -race ./...
```

The [integration module](integration/README.md) keeps Vine out of the library's
module graph. CI runs both suites. See [AGENTS.md](AGENTS.md) for repository boundaries.
