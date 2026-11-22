# vrpc-go

[![License](https://img.shields.io/github/license/yorun-ai/vrpc-go)](LICENSE)
[![Version](https://img.shields.io/github/v/release/yorun-ai/vrpc-go?label=version&cacheSeconds=300)](https://github.com/yorun-ai/vrpc-go/releases/latest)
[![Go](https://img.shields.io/github/go-mod/go-version/yorun-ai/vrpc-go)](go.mod)
[![Go Reference](https://pkg.go.dev/badge/go.yorun.ai/vrpc.svg)](https://pkg.go.dev/go.yorun.ai/vrpc)
[![CI](https://github.com/yorun-ai/vrpc-go/actions/workflows/ci.yml/badge.svg)](https://github.com/yorun-ai/vrpc-go/actions/workflows/ci.yml)

A standalone Go client for [vRPC](https://github.com/yorun-ai/vrpc-ts), compatible with [Vine Portal](https://github.com/yorun-ai/vine).

**English** | [简体中文](README.zh-CN.md)

- Module `go.yorun.ai/vrpc`, package `vrpc`.
- Requires Go 1.27 or later. JSON and CBOR/Binary encoding are built in. CBOR uses fxamacker/cbor.
- The client does not depend on Vine, the application lifecycle, dependency injection, an actor registry, or a framework logger.
- Apache License 2.0. The public API is still stabilizing before v1.0.0.

## Getting started

The module path `go.yorun.ai/vrpc` resolves to this repository. Install it with:

```sh
go get go.yorun.ai/vrpc@v0.17.0
```

The repository can be cloned from [yorun-ai/vrpc-go](https://github.com/yorun-ai/vrpc-go) and tested with `go test ./...`.
To use an unpublished checkout, add a local replacement in the consuming module:

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
        InstanceID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174001"),
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
result, metadata, err := client.InvokeAs[User](ctx, methodInfo,
    struct { ID int64 `json:"id"` }{ID: 42})
```

`client.InvokeAs[T]` returns the decoded result, response metadata, and an error.
Methods without a result use `client.InvokeAs[struct{}]`.

`client.InvokeRaw(ctx, service, method, params, options...)` invokes an unregistered
method with JSON and returns `(any, *ResponseMetadata, error)`. JSON objects decode
to `map[string]any`, arrays to `[]any`, and numbers to `float64`.

On failure, `InvokeAs[T]` returns the zero value of `T` and `InvokeRaw` returns nil.
Response metadata remains available when an HTTP response was received.

The example assumes that `context`, `fmt`, `uuid`, and `go.yorun.ai/vrpc` are imported,
and that `ctx` and `token` are defined by the caller. Service, method, and field names
must match the server contract. `Endpoint` is the invocation URL prefix and includes
`/invoke` when it is part of the route; the client appends `/<service>/<method>`.

The repository also provides a command-line example:

```sh
# Set PORTAL_KEY in the environment for authenticated calls.
go run ./examples/portal -endpoint https://api.example.com/invoke \
  -client-name demo.client -client-version 1.0.0 \
  -client-instance-id 123e4567-e89b-12d3-a456-426614174001 \
  -service demo.UserService -method Get -params '{"id":42}'
```

## Configuration and behavior

`Identity.Name` requires lowercase letters separated by dots, and `Identity.Version`
requires a complete semantic version. Each client should use a stable
`Identity.InstanceID`.

Each call creates a trace and span. `vrpc.WithTrace(parent)` preserves the parent
trace ID and creates a child span.

`Option.Timeout` defaults to 30 seconds. `vrpc.WithTimeout(duration)` overrides it for
one call, and an earlier context deadline takes precedence. The remaining duration is
sent in `vrpc-options`. The current Portal defaults to 30 seconds and rejects values
above 120 seconds; this client does not hardcode that gateway limit.

Canceling a call stops the client from waiting, but Portal may continue execution until
its own deadline. The client does not retry invocations, follow redirects, add an
idempotency header, or synthesize framework Actor and Initiator values. The caller is
responsible for a custom transport's behavior.

Fixed credentials are set through `Option.Authorization` and encoded once when the
client is created; later changes to the map do not affect the client. A non-nil map
overrides `Headers["Authorization"]`; an empty map removes that header, and nil leaves
the custom header unchanged. `EncodeCredentials` formats Portal fields as
`field value, field value` in sorted order and rejects commas, control characters,
empty values, and case-insensitive duplicate field names. It neither assumes Bearer
authentication nor authenticates requests locally.

`Headers` accepts additional HTTP headers and preserves multiple values; it is copied
at construction. Client-generated headers override custom values with the same name.
`Transport` accepts an `http.RoundTripper` for custom TLS or networking, and the caller
owns its lifecycle. A client supports concurrent calls when its transport is safe for
concurrent use.

## Client registry

`Registry` stores client service and method contracts. It does not provide server
registration, executors, dependency injection, or service discovery. Registration
copies method descriptors, rejects duplicates, and publishes a service atomically;
concurrent registration and lookup are supported.

Generated client packages and hand-written callers use the following registration API:

```go
// In a generated client package that imports go.yorun.ai/vrpc.
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

Look up the `MethodInfo` once with `GetMethodInfo`, check the returned boolean, and pass
the descriptor directly to `client.InvokeAs[T](ctx, methodInfo, params, options...)`.
A generated client can retain the descriptor after package initialization. The
descriptor contains the service name, method name, invocation path, and binary flags.
Invocations do not consult the registry, and an empty descriptor is rejected before a
request is sent.

When a missing method should be treated as an initialization error, use
`methodInfo := vrpc.MustGetMethodInfo("demo.Files", "upload")` instead; `Registry` also
provides `registry.MustGetMethodInfo`. Both functions panic with the service and method
names when the contract is missing.

Request and response encoding are selected independently. Binary arguments use CBOR,
and binary results advertise CBOR and JSON in `Accept`; other arguments and results use
JSON. Both codecs are built in and require no codec imports or configuration.

`ServiceSpec.Name` and `MethodSpec.Name` store Go names, which are exposed through
`MethodInfo.ServiceName()` and `Name()`. Routing continues to use `SkelName`.

`MethodSpec.ArgumentsSensitive` and `ResultSensitive` mark sensitive payloads. The
matching `MethodInfo` accessors expose these flags to logging and diagnostics; the
flags do not affect encoding or redact payloads automatically.

Use `NewRegistry` for an isolated contract set, or use the package-level `Register` and
`GetMethodInfo` for the default registry. A client can use a descriptor from either
registry without additional configuration. `Register` returns no value and panics on
invalid or duplicate registration.

skelc's Go API client generator targets this standalone runtime; generated backend
packages use Vine.

## JSON and Binary

JSON is the default encoding. Typed structs with `json` tags preserve Go integer
precision. An `any` or map result follows standard JSON decoding rules; use typed
fields when integer precision matters. Under `encoding/json/v2`, nil slices and maps
encode as empty collections, matching current Vine behavior.

For binary arguments or results, register the method's binary flags as shown above.
`[]byte` values use CBOR byte strings, integer map keys remain integers, and nil
collections encode as empty collections. Use matching `json` and `cbor` field names.
When no `cbor` tag is present, fxamacker/cbor falls back to `json` tags. Methods with
binary results still accept JSON responses, including Portal errors. Calls that
advertise only JSON in `Accept` reject CBOR responses. Compression is handled by the
transport, and the default Go transport
negotiates and decodes gzip. This package does not advertise zstd support.

## Errors and metadata

A call succeeds only when the response has HTTP 2xx status, its required headers
(`content-type`, `vrpc-status`, and `vrpc-server`) are valid, `vrpc-status` is `OK`, and
the envelope contains a result but no error. Unrecognized uppercase status codes remain
available in the metadata for forward compatibility.

- `*vrpc.InvocationError` preserves a failing remote status or non-2xx HTTP response,
  an optional `ErrorPayload`, and any decoding failure in `Cause`.
- `*vrpc.ProtocolError` identifies a malformed successful response.

Transport and context errors wrap their causes. Use `errors.Is` for cancellation and
deadline errors and `errors.As` for structured errors.

`vrpc.ErrorPayload` is an alias of the shared `transport/http.ErrorPayload`, which
contains `Code`, `Message`, `Reason`, and `Detail`.

`ResponseMetadata` exposes HTTP status, protocol status, server identity, and response
headers. Read `portal-trace-id` from `ResponseMetadata.Header` when needed, and
classify remote outcomes by `ResponseMetadata.Status` rather than by message text or
the auxiliary `Code` field. The shared transport limits response bodies to 128 MiB
after decompression. Protocol error messages do not include the response body.

## Shared transport

`transport/http` provides protocol constants, header validation, timeout handling,
JSON/CBOR request and response codecs, body limits, and the HTTP exchange lifecycle.
Adapters pass Go values to `EncodeRequest` and `EncodeResponse`, representing errors
as `*ErrorPayload`. They decode arguments with `DecodeRequest` and use
`DecodeResponse` with `ResponsePayload.Unmarshal` for results or
`ResponsePayload.DecodeError` for errors. These operations follow fixed wire rules:
nil collections encode as empty collections, and decoding rejects duplicate keys.
Adapters retain transport configuration, application metadata, and error policy. Raw
envelope helpers preserve encoded payloads; `ResponsePayload.EncodeWithError` replaces
an error without decoding and re-encoding the result.

`ReadRequestBody` limits the complete encoded request envelope to 32 MiB for JSON and
128 MiB for CBOR, selected by `Content-Type`. It checks both the declared length and
the bytes read, including requests with no declared length. The response limit remains
128 MiB. `RequestBodyLimit` returns the limit for an encoding. `MaxRequestBodyBytes`
has been removed; callers that used it should switch to `MaxJSONRequestBodyBytes` or
`MaxCBORRequestBodyBytes`. These limits are fixed by the transport and are not client
options. A receiving server must use the updated `transport/http` to accept CBOR
requests larger than 32 MiB.

## Compatibility and scope

Tests cover client invocation, headers, errors, cancellation, and the shared JSON/CBOR
wire implementation. Portal integration and gateway policy tests belong to Vine.

The client supports hand-written request and result types and skelc-generated Go API
clients. Generated backend service clients use Vine. Vine consumes the shared transport
through its framework adapter. The client does not implement Portal `/inspect`.

## Package layout

The root `api.go` exposes the client API through type aliases and function wrappers.
Client, registry, credential, and protocol metadata implementations live in `internal`,
while consumers continue to import `go.yorun.ai/vrpc`. `transport/http` remains public
for cross-module reuse by Vine.

## Development

```sh
GOWORK=off go test -race ./...
GOWORK=off go vet ./...
```

CI runs the client and shared transport tests without a Vine dependency.
See [AGENTS.md](AGENTS.md) for repository boundaries.

## Skel scalar types

The `go.yorun.ai/skel/types` package (imported as `skeltype`) provides Decimal, Binary,
Timestamp, Duration, LocalDate, LocalTime, LocalDateTime, and JSON types and their
constructors. vRPC reuses these types and their JSON/CBOR encodings; the separate
`go.yorun.ai/vrpc/skel` package is no longer provided. Their representations, including
decimal scale and string-based date/time values, match Vine contracts. Generated API
clients and hand-written callers use the same types without depending on Vine.

The shared `go.yorun.ai/skel/types/codec` encodes UUID values and map keys as
strings in JSON and CBOR. It supports both `uuid.UUID` and `skeltype.UUID`.
