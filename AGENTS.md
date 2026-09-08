# Go vRPC client guidelines

- This repository owns the standalone Go vRPC client, module `go.yorun.ai/vrpc`.
- Keep the root package standard-library-only. Optional CBOR support lives in `codec/cbor`.
- `transport/http` owns the wire implementation extracted from Vine; both clients and framework adapters must use it. Keep schema encoding and framework metadata in the adapter.
- Never import Vine. Check wire compatibility against Vine and vrpc-ts when changing the protocol.
- Use Go 1.27, `encoding/json/v2`, `Rpc` in identifiers, and `_` prefixes for unexported production types.
- Public APIs need GoDoc. Preserve context cancellation and deadlines; do not add invocation retries.
- Keep tests beside implementation, with filesystem/network resources owned by the test.
- Keep README.md and README.zh-CN.md synchronized. Integration tests live in a separate module so Vine stays out of the client dependency graph.
- Run gofmt, GOWORK=off go test -race ./..., GOWORK=off go vet ./..., and git diff --check.
- Run the separate integration module when changing wire behavior. Only the integration module may replace `go.yorun.ai/vrpc` with `..`; do not commit machine-specific or Vine source replacements.
