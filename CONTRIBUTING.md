# Contributing

Read AGENTS.md and the README before changing the client. Preserve the standalone dependency boundary and verify wire changes against the integration module. Keep public API documentation and both READMEs synchronized.

Run gofmt, `GOWORK=off go test -race ./...`, `GOWORK=off go vet ./...`, and `git diff --check` in the root. Run the same Go tests and vet in `integration/` for protocol changes. JSON client dependencies must remain standard-library-only; CBOR dependencies are isolated to `codec/cbor`.

Before a first release, configure the `go.yorun.ai/vrpc` vanity import mapping to `https://github.com/yorun-ai/vrpc-go`, verify imports from a clean external module, and create a reviewed version tag. Root API design remains provisional until that release. Do not publish the integration module.

Contributions are licensed under Apache-2.0.
