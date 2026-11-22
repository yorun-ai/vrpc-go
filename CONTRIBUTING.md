# Contributing

Read AGENTS.md and both READMEs before changing the client. Preserve the standalone
dependency boundary, and verify wire changes with the client and shared transport
tests. Portal integration and gateway policy tests belong to Vine. Keep the public API
documentation and both READMEs synchronized.

Run gofmt, `GOWORK=off go test -race ./...`, `GOWORK=off go vet ./...`, and
`git diff --check` from the repository root. JSON and CBOR are built in and selected
per method from the client registry; both use the shared transport envelope
implementation.

Before publication, verify imports from a clean external module and create a reviewed
version tag. The `go.yorun.ai/vrpc` vanity import resolves to
`https://github.com/yorun-ai/vrpc-go`.

Contributions are licensed under Apache-2.0.

## Release workflow

Merge feature and fix PRs first. Then open a separate release-preparation PR with a
dated `## [X.Y.Z] - YYYY-MM-DD` CHANGELOG entry covering the merged changes since the
previous tag, plus any release version references. After the required CI passes and the
PR is merged, sync main and push `vX.Y.Z` from the reviewed commit. The tag triggers
`.github/workflows/release.yml`, which validates the checkout, main ancestry, and notes,
runs standalone race tests and vet, and then creates and publishes a Draft Release. No
CLI archives are built. The Go module is distributed by tag and is available
independently of GitHub Release publication.

A failed run can be retried or manually dispatched with the existing tag. Published
Releases remain unchanged, and drafts can be resumed. Version tags are never moved.
Verifying hosted publication permissions requires an actual tag run.

For workflow changes, run `bash .github/scripts/release_test.sh`,
`shellcheck .github/scripts/*.sh`, actionlint, and `git diff --check`.
