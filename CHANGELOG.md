# Changelog

## [Unreleased]

## [0.14.0] - 2026-10-07

### Added

- Expose fixed JSON/CBOR request and response codecs through `transport/http`:
  `EncodeRequest`, `DecodeRequest`, `EncodeResponse` and `DecodeResponse`.
  Clients and framework adapters share the same payload and envelope rules.
- Expose the shared `transport/http.ErrorPayload` and response helpers
  `DecodeError` and `EncodeWithError`. Error replacement preserves the encoded
  result without decoding and re-encoding it.

### Changed

- Reuse portable scalar values and encodings from `go.yorun.ai/skel/types`,
  depending on the released Skel v0.31.0 module.
- Make `vrpc.ErrorPayload` an alias of the shared transport payload, containing
  `Code`, `Message`, `Reason` and `Detail`.
- Make `ResponsePayload.Unmarshal` a method using fixed wire rules, replacing
  the configurable decoder field.
- Apply duplicate-key rejection to CBOR request/response envelopes as well as
  decoded payloads. JSON and CBOR continue to encode nil collections as empty
  collections.

### Removed

- Remove the `go.yorun.ai/vrpc/skel` package and its duplicate scalar types.
- Remove the obsolete error payload `Type` field and the custom decoder hook
  `DecodeCBORResponseWith`.
- Remove the Portal integration test module and its CI gate. Portal integration
  and gateway policy tests belong to Vine; client and shared protocol tests
  remain in this repository, including race and vet checks.

### Maintenance

- Publish releases from reviewed version tags through the release workflow,
  validating tag identity, main ancestry, release notes and standalone checks.
- Enforce the standalone dependency boundary for both production and test code.

### Upgrade Notes

- Replace imports of `go.yorun.ai/vrpc/skel` with `go.yorun.ai/skel/types`, using
  the `skeltype` import alias. Regenerate Go API clients with the matching Skel
  toolchain and update hand-written scalar references.
- Stop reading or setting `ErrorPayload.Type`. Use the response metadata status
  to classify invocation outcomes; error payload `Code` remains auxiliary data.
- Use `DecodeCBORResponse` or `DecodeResponse` instead of supplying a custom
  response decoder. Obtain `ResponsePayload` from the decoder and call its
  `Unmarshal` or `DecodeError` methods for typed values.
- Framework adapters can use the shared value codecs and `*ErrorPayload` rather
  than maintaining separate JSON/CBOR and error-envelope implementations. The
  standalone client API remains independent of Vine.

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

[Unreleased]: https://github.com/yorun-ai/vrpc-go/compare/v0.14.0...HEAD
[0.14.0]: https://github.com/yorun-ai/vrpc-go/compare/v0.13.0...v0.14.0
[0.13.0]: https://github.com/yorun-ai/vrpc-go/compare/v0.12.0...v0.13.0
