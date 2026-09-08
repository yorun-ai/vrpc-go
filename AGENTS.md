# Go vRPC client guidelines

- This repository owns the standalone Go vRPC client, module `go.yorun.ai/vrpc`.
- Keep root `api.go` a public facade over `internal`. Public types may use aliases; callable APIs must use function declarations, never function-valued package variables.
- Keep `transport/http` and its CBOR envelopes public so Vine can reuse them across module boundaries.
- JSON and CBOR are built-in client capabilities. Select encoding from registered method binary flags; do not expose codec configuration or a separate codec package.
- `transport/http` owns the shared vRPC wire implementation; both clients and framework adapters must use it. Keep schema encoding and framework metadata in the adapter.
- Never import Vine. Check wire compatibility against Vine and vrpc-ts when changing the protocol.
- Use Go 1.27, `encoding/json/v2`, `Rpc` in identifiers, and `_` prefixes for unexported production types.
- Write function bodies and non-empty struct declarations across multiple lines, including short facade wrappers and getters. Expand all non-empty struct literals with one field per line, including single-field literals and tests.
- Keep concise GoDoc on public facade and transport APIs. Avoid comments that restate internal names or obvious operations; retain non-obvious behavior and constraints. Preserve context cancellation and deadlines; do not add invocation retries.
- Keep tests beside implementation, with filesystem/network resources owned by the test.
- Keep README.md and README.zh-CN.md synchronized. Integration tests live in the separate `test/integration` module so Vine stays out of the client dependency graph.
- Run gofmt, GOWORK=off go test -race ./..., GOWORK=off go vet ./..., and git diff --check.
- Run the separate integration module when changing wire behavior. Only the integration module may replace `go.yorun.ai/vrpc` with `../..`; do not commit machine-specific or Vine source replacements.

- Separate logical phases in long functions with blank lines; keep each operation and its error check together.
