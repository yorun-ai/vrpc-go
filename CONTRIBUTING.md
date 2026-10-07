# Contributing

Read AGENTS.md and the README before changing the client. Preserve the standalone dependency boundary and verify wire changes with the client and shared transport tests. Portal integration and gateway policy tests belong to Vine. Keep public API documentation and both READMEs synchronized.

Run gofmt, `GOWORK=off go test -race ./...`, `GOWORK=off go vet ./...`, and `git diff --check` in the root. JSON and CBOR are built in, with method-level selection from the client registry. Keep both on the shared transport envelope implementation.

Before publication, verify imports from a clean external module and create a reviewed version tag. The `go.yorun.ai/vrpc` vanity import maps to `https://github.com/yorun-ai/vrpc-go`.

Contributions are licensed under Apache-2.0.

## Release workflow

Merge feature and fix PRs first. Then open a separate release-preparation PR
with a dated `## [X.Y.Z] - YYYY-MM-DD` CHANGELOG entry covering the merged
changes since the previous tag, plus any release version references.
After required CI passes and the PR is merged, sync main and push `vX.Y.Z`
from the reviewed commit. The tag triggers `.github/workflows/release.yml`:
validate the checkout, main ancestry and notes, run standalone race tests and vet,
then create a Draft Release and publish it. No CLI archives are built; the Go
module is distributed by tag and does not wait for GitHub Release publication.

A failed run can be retried or manually dispatched with the existing tag.
Existing published Releases are left unchanged; drafts can resume publication.
Never move a version tag. A real tag run is
needed to verify hosted publication permissions.

For workflow changes, run `bash .github/scripts/release_test.sh`,
`shellcheck .github/scripts/*.sh`, actionlint and `git diff --check`.
