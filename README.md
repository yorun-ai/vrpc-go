# vrpc-go

A standalone Go client for [vRPC](https://github.com/yorun-ai/vrpc-ts), compatible with [Vine Portal](https://github.com/yorun-ai/vine).

**English** | [简体中文](README.zh-CN.md)

- Module: `go.yorun.ai/vrpc`; package: `vrpc`.
- Go 1.27 or later. Built-in JSON and CBOR/Binary support (fxamacker/cbor).
- No dependency on Vine, application lifecycle, DI, actor registry, or framework logger.
- Apache License 2.0. Initial implementation; the public API is not yet versioned.

## Try the client

Clone [yorun-ai/vrpc-go](https://github.com/yorun-ai/vrpc-go) and run `go test ./...`.
The import path `go.yorun.ai/vrpc` resolves to this repository. Until the first version is published, use a local checkout from a consuming module:

```sh
go mod edit -require=go.yorun.ai/vrpc@v0.0.0
go mod edit -replace=go.yorun.ai/vrpc=/absolute/path/to/vrpc-go
go mod tidy
```

```go
client, err := vrpc.NewClient(vrpc.Option{
    Endpoint: "https://api.example.com/invoke",
    Identity: vrpc.Identity{
        Name: "demo.client",
        Version: "1.0.0",
        InstanceID: "123e4567-e89b-12d3-a456-426614174001",
    },
    Authorization: map[string]string{
        "key": token,
    },
})
if err != nil {
    return err
}

registry := vrpc.NewRegistry()
registry.Register(&vrpc.ServiceSpec{
    SkelName: "demo.UserService",
    Methods: []vrpc.MethodSpec{
        {
            SkelName: "Get",
        },
    },
})
methodInfo, ok := registry.GetMethodInfo("demo.UserService", "Get")
if !ok {
    return fmt.Errorf("method not registered")
}

type User struct {
    Name string `json:"name"`
}
result, metadata, err := client.Invoke[User](ctx, methodInfo,
    struct { ID int64 `json:"id"` }{ID: 42})
```

`client.Invoke[T]` returns the decoded result, response metadata and an error.
Use `client.Invoke[struct{}]` for methods without a result.

`client.InvokeRaw(ctx, service, method, params, options...)` invokes an unregistered
method using JSON and returns `(any, *ResponseMetadata, error)`. JSON objects decode
to `map[string]any`, arrays to `[]any`, and numbers to `float64`.

On failure, `Invoke[T]` returns the zero value of `T` and `InvokeRaw` returns nil; metadata remains available
when an HTTP response was received.

Import `context`, `fmt` and `go.yorun.ai/vrpc`. The example uses the caller's `ctx` and `token`.
Use your service's exact contract names. `Endpoint` includes the invocation prefix;
the client appends `/<service>/<method>` and does not automatically add `/invoke`.

A complete command-line example is included:

```sh
# Set PORTAL_KEY in your environment for authenticated calls.
go run ./examples/portal -endpoint https://api.example.com/invoke \
  -client-name demo.client -client-version 1.0.0 \
  -client-instance-id 123e4567-e89b-12d3-a456-426614174001 \
  -service demo.UserService -method Get -params '{"id":42}'
```

## Configuration and behavior

`Identity.Name`, `Identity.Version` and `Identity.InstanceID` are required.
Missing or invalid values cause client construction to fail.
Use lowercase letters separated by dots for the name and a full semantic version.
The instance UUID is stable for that client. Each call creates a trace and span;
`vrpc.WithTrace(parent)` preserves a parent trace ID and creates a child span.

`Option.Timeout` defaults to 30 seconds. `vrpc.WithTimeout(duration)` overrides
it for a call; an earlier context deadline takes precedence. The remaining
duration is sent in `vrpc-options`. Current Portal defaults to 30 seconds and
rejects values above 120 seconds. The client does not hardcode that gateway limit.

Client cancellation stops waiting, but Portal may continue execution until its
own deadline. The client does not add invocation retries or follow redirects.
Custom transports are responsible for their own behavior. No idempotency header
is added. Framework Actor and Initiator values are not synthesized by the client.

Set fixed credentials through `Option.Authorization`. They are encoded once when
the client is created; later changes to the map do not affect it. A non-nil map
overrides `Headers["Authorization"]`; an empty map removes that header, while nil
leaves the custom header unchanged. `EncodeCredentials` formats
Portal fields as `field value, field value` in sorted order; it rejects commas,
control characters, empty values, and case-insensitive duplicate field names.
It does not assume Bearer authentication or authenticate locally.

`Headers` accepts additional HTTP headers, preserving multiple values, and is copied at construction.
Client-generated headers override custom values with the same name. `Transport` accepts
an `http.RoundTripper` for custom TLS or networking; the caller owns its lifecycle.
Clients support concurrent calls when their custom transport does too.

## Client registry

`Registry` stores client service and method contracts. It has no server registration,
executors, DI or service discovery.
Registration copies the method descriptors, rejects duplicates and publishes a service
atomically; concurrent calls and registration are supported.

This is the registration API intended for generated client packages:

```go
// In a generated client package; imports go.yorun.ai/vrpc.
func init() {
    vrpc.Register(&vrpc.ServiceSpec{
        SkelName: "demo.Files",
        Methods: []vrpc.MethodSpec{
            {SkelName: "upload", ArgumentsContainsBinaryType: true},
            {SkelName: "download", ResultContainsBinaryType: true},
        },
    })
}
```

Resolve `MethodInfo` once with `GetMethodInfo`, check the returned boolean, and pass
the descriptor directly to `client.Invoke[T](ctx, methodInfo, params, options...)`.
Generated clients can retain it after package initialization. A descriptor contains
the service name, method name, invocation path and binary flags. Invocations do not
look up the registry; an empty descriptor is rejected before sending a request.

Request and response encoding are selected independently: binary arguments use CBOR;
binary results advertise CBOR and JSON in `Accept`. Other arguments/results use JSON.
Both codecs are built in, with no codec imports or configuration required.

`ServiceSpec.Name` and `MethodSpec.Name` store Go names, exposed through
`MethodInfo.ServiceName()` and `Name()`. Routing continues to use `SkelName`.

`MethodSpec.ArgumentsSensitive` and `ResultSensitive` mark sensitive payloads.
Their corresponding `MethodInfo` accessors let logging and diagnostics inspect
these flags; they do not change encoding or automatically redact payloads.

Use `NewRegistry` for isolated contracts, or package-level `Register`/`GetMethodInfo`
for the default registry. Clients can invoke descriptors from either without registry
configuration. `Register` has no return value and panics on invalid or duplicate
registration.

The registry and invocation support are implemented here; skelc's existing Go generator
still targets Vine and needs a separate adaptation to emit these registrations.

## JSON and Binary

JSON is the default. Typed structs with `json` tags preserve Go integer precision.
An `any`/map result follows standard JSON decoding rules; use typed fields when
integer precision matters. Nil slices/maps encode as empty collections under
`encoding/json/v2`, matching current Vine encoding. Hand-written callers also register a method contract and obtain its descriptor.

For Binary arguments/results, register the method's binary flags as shown above.
`[]byte` uses CBOR byte strings, integer map keys remain integers, and nil collections
encode as empty collections. Use matching `json` and `cbor` field names; fxamacker also
falls back to `json` tags. JSON responses remain accepted for binary-result methods,
including Portal errors. Calls that only advertise JSON reject CBOR responses.
Compression is handled
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
- Transport/context errors wrap their causes; use `errors.Is` for
  cancellation/deadline errors and `errors.As` for structured errors.

`ResponseMetadata` exposes HTTP status, protocol status, server identity and response headers.
Read `portal-trace-id` from `ResponseMetadata.Header` when needed. Classify remote outcomes by `ResponseMetadata.Status`,
not message text or the auxiliary payload's `Code`. The shared transport limits response bodies
to 128 MiB after transport decompression. No response body is
included in protocol error messages.

## Shared transport

`transport/http` provides wire constants, header checks, timeout handling,
raw JSON/CBOR envelopes, body limits and the HTTP exchange lifecycle.
JSON and CBOR envelopes are provided by the same package.
Adapters supply encoded arguments and results, transport configuration and
application-specific metadata and error handling. The shared envelopes preserve
encoded payloads unchanged, including collection encoding profiles.

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

## Package layout

The root `api.go` exposes the client API through type aliases and ordinary function
wrappers. Client, registry, codecs, credentials and protocol metadata are implemented
in `internal`; consumers continue to import `go.yorun.ai/vrpc`.
`transport/http` remains public for cross-module reuse by Vine.

## Development

```sh
GOWORK=off go test -race ./...
GOWORK=off go vet ./...
cd test/integration
GOWORK=off go test -race ./...
```

The [integration module](test/integration/README.md) keeps Vine out of the library's
module graph. CI runs both suites. See [AGENTS.md](AGENTS.md) for repository boundaries.
