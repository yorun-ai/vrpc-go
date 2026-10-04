# Changelog

## [Unreleased]

## [0.13.0] - 2026-10-04

### Added

- Add `vrpc.MustGetMethodInfo` and `Registry.MustGetMethodInfo` for required method lookups during client initialization. Missing methods panic with the service and method names; successful lookups retain all registered metadata.

### Maintenance

- Remove duplicate main-branch CI runs while retaining client and Portal integration checks on pull requests.

### Upgrade Notes

- This release is additive: existing lookup, registration, invocation and wire behavior remain compatible. Update the runtime dependency before adopting the new helper in generated or hand-written clients.

## 0.12.0 - 2026-09-09

- Rename the generic client method `Invoke[T]` to `InvokeAs[T]`; update typed calls to use the new name. `InvokeRaw` is unchanged.

## 0.11.0 - 2026-09-09

- Add `go.yorun.ai/vrpc/skel` with portable scalar types, constructors, and Vine-compatible JSON/CBOR encoding for generated API clients.

## 0.10.0 - 2026-09-09

- Initial standalone Go vRPC client with Portal credentials, deadlines, trace propagation and structured errors.
- Generic `Invoke[T]` returns typed results, response metadata and errors; JSON `InvokeRaw` calls services by name without registration.
- Client-only registry retains Go names, sensitive flags and binary flags in immutable method descriptors. Invalid or duplicate registrations panic.
- Built-in JSON and CBOR encoding selected from method contracts.
- Public HTTP transport helpers cover request/response envelopes, headers, body limits and round-trip lifecycle for framework adapters.
- Root API facade over internal client implementation, with no Vine production dependency.
- Raw Portal CLI example and isolated Portal compatibility tests covering typed and raw invocations.
- README badges for license, version, Go, package reference and CI.

[Unreleased]: https://github.com/yorun-ai/vrpc-go/compare/v0.13.0...HEAD
[0.13.0]: https://github.com/yorun-ai/vrpc-go/compare/v0.12.0...v0.13.0
