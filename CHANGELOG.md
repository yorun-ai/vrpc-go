# Changelog

## Unreleased

- Return result, metadata and error together from `Invoke[T]`; add JSON `InvokeRaw` for unregistered service/method calls.

- Name response metadata `ResponseMetadata`, expose it through `Invoke` and error `Metadata` fields.
- Allow codec dependencies in CI while rejecting dependencies on Vine.

- Invoke registered `MethodInfo` descriptors directly; cache invocation paths and remove client registry lookup/configuration.

- Return typed business results through generic client methods `Invoke[T]`; remove the public result-pointer invocation method.

- Move client implementation into `internal` behind the root `api.go` facade; retain public shared transport packages.

- Make JSON/CBOR codecs internal and automatic from registry binary flags; remove the codec subpackage and public codec configuration.

- Add a client-only service/method registry with atomic registration, isolated registries and method-level JSON/CBOR negotiation.

- Provide shared `transport/http` and CBOR envelopes for clients and framework adapters. Preserve schema-specific raw payloads, existing wire diagnostics, and transport/decode error boundaries.

- Initial standalone vRPC HTTP client with JSON, CBOR/Binary, Portal credentials, deadlines, trace propagation, structured errors, and response metadata.
- Portal v0.15.3 compatibility tests in an isolated integration module.
