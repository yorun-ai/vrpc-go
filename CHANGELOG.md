# Changelog

## Unreleased

## 0.10.0 - 2026-09-09

- Initial standalone Go vRPC client with Portal credentials, deadlines, trace propagation and structured errors.
- Generic `Invoke[T]` returns typed results, response metadata and errors; JSON `InvokeRaw` calls services by name without registration.
- Client-only registry retains Go names, sensitive flags and binary flags in immutable method descriptors. Invalid or duplicate registrations panic.
- Built-in JSON and CBOR encoding selected from method contracts.
- Public HTTP transport helpers cover request/response envelopes, headers, body limits and round-trip lifecycle for framework adapters.
- Root API facade over internal client implementation, with no Vine production dependency.
- Raw Portal CLI example and isolated Portal compatibility tests covering typed and raw invocations.
- README badges for license, version, Go, package reference and CI.
