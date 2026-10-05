# Contributing

Read AGENTS.md and the README before changing the client. Preserve the standalone dependency boundary and verify wire changes against the integration module. Keep public API documentation and both READMEs synchronized.

Run gofmt, `GOWORK=off go test -race ./...`, `GOWORK=off go vet ./...`, and `git diff --check` in the root. Run the same Go tests and vet in `test/integration/` for protocol changes. JSON and CBOR are built in, with method-level selection from the client registry. Keep both on the shared transport envelope implementation.

Before a first release, verify the `go.yorun.ai/vrpc` vanity import mapping to `https://github.com/yorun-ai/vrpc-go`, verify imports from a clean external module, and create a reviewed version tag. Root API design remains provisional until that release. Do not publish the integration module.

Contributions are licensed under Apache-2.0.

## Release workflow

Prepare a dated `## [X.Y.Z] - YYYY-MM-DD` CHANGELOG entry in the release PR.
After required CI passes and the PR is merged, sync main and push `vX.Y.Z`
from the reviewed commit. The tag triggers `.github/workflows/release.yml`:
validate the checkout, main ancestry and notes, run standalone race tests and vet,
then create a Draft Release and publish it. No CLI archives are built; the Go
module is distributed by tag and does not wait for GitHub Release publication.

A failed run can be retried or manually dispatched with the existing tag.
Existing published Releases are left unchanged; drafts can resume publication.
Never move a version tag or release the integration module. A real tag run is
needed to verify hosted publication permissions.

For workflow changes, run `bash .github/scripts/release_test.sh`,
`shellcheck .github/scripts/*.sh`, actionlint and `git diff --check`.
