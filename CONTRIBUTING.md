# Contributing

Read AGENTS.md and the README before changing the client. Preserve the standalone dependency boundary and verify wire changes against the integration module. Keep public API documentation and both READMEs synchronized.

Run gofmt, `GOWORK=off go test -race ./...`, `GOWORK=off go vet ./...`, and `git diff --check` in the root. Run the same Go tests and vet in `test/integration/` for protocol changes. JSON and CBOR are built in, with method-level selection from the client registry. Keep both on the shared transport envelope implementation.

Before a first release, verify the `go.yorun.ai/vrpc` vanity import mapping to `https://github.com/yorun-ai/vrpc-go`, verify imports from a clean external module, and create a reviewed version tag. Root API design remains provisional until that release. Do not publish the integration module.

Contributions are licensed under Apache-2.0.
