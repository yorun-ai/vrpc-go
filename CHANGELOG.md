# Changelog

## Unreleased

- Extract Vine's HTTP transport into shared `transport/http` and optional CBOR envelopes; both the standalone client and Vine adapters reuse the implementation. Preserve schema-specific raw payloads, existing wire diagnostics, and transport/decode error boundaries.

- Initial standalone vRPC HTTP client with JSON, optional CBOR/Binary, Portal credentials, deadlines, trace propagation, structured errors, and response metadata.
- Portal v0.15.3 compatibility tests in an isolated integration module.
