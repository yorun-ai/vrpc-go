# Go vRPC client guidelines

- This repository owns the standalone Go vRPC client, module `go.yorun.ai/vrpc`.
- Keep root `api.go` a public facade over `internal`. Public types may use aliases; callable APIs must use function declarations, never function-valued package variables.
- Keep `transport/http` and its CBOR envelopes public so Vine can reuse them across module boundaries.
- JSON and CBOR are built-in client capabilities. Select encoding from registered method binary flags; do not expose codec configuration or a separate codec package.
- `transport/http` owns the shared vRPC wire implementation; both clients and framework adapters must use it. Use its fixed request/response codecs for payloads and envelopes; keep framework metadata and error policy in the adapter.
- Reuse `go.yorun.ai/skel/types` (import alias `skeltype`) for Skel scalar values and their JSON/CBOR encodings. Do not reintroduce local scalar definitions or forwarding packages. Client registry specs describe invocation metadata, independently of full language descriptors.
- Never import Vine, including in tests. Keep the shared wire contract aligned with vrpc-ts; Vine owns Portal integration and gateway policy tests.
- Use Go 1.27, `encoding/json/v2`, `Rpc` in identifiers, and `_` prefixes for unexported production types.
- Write function bodies and non-empty struct declarations across multiple lines, including short facade wrappers and getters. Expand all non-empty struct literals with one field per line, including single-field literals and tests.
- Keep concise GoDoc on public facade and transport APIs. Avoid comments that restate internal names or obvious operations; retain non-obvious behavior and constraints. Preserve context cancellation and deadlines; do not add invocation retries.
- Keep tests beside implementation, with filesystem/network resources owned by the test.
- Keep README.md and README.zh-CN.md synchronized. Test client behavior and the shared wire implementation within this repository.
- Run gofmt, GOWORK=off go test -race ./..., GOWORK=off go vet ./..., and git diff --check.
- Do not commit machine-specific or Vine source replacements.

- Separate logical phases in long functions with blank lines; keep each operation and its error check together.

## Release Publication

- Merge implementation changes through their own PRs before opening a separate release-preparation PR. Derive its CHANGELOG from the merged changes since the previous tag; do not combine feature changes and release preparation.
- Prepare the dated CHANGELOG entry in a release PR, pass CI and merge, sync main, then push the reviewed version tag. A `v*` tag triggers the Release workflow.
- Validate tag identity and main ancestry, run standalone tests and vet, then create GitHub Release using the corresponding CHANGELOG notes. No binary archives are produced.
- Retry or manually dispatch with the same existing tag after failure. Published Releases are left unchanged; do not move tags. A Go module is available through its tag independently of GitHub Release, so tag publication is not reversible through Release cleanup.
- See CONTRIBUTING.md for checks and publication recovery.
