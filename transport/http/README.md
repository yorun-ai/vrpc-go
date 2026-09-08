# Shared vRPC HTTP transport

This package is extracted from Vine's `internal/core/rpc/transport/http` (Apache-2.0), not a second protocol implementation.

- `proto.go`: wire constants, header/media validation, delimited metadata, timeout and path parsing.
- `body.go`: bounded request/response body reads.
- `envelope.go` and `cbor/`: raw request/response envelopes. Arguments and results arrive already encoded so Vine's per-schema encoder and legacy collection profile remain authoritative.
- `exchange.go`: request construction, prepared callback, one HTTP exchange and response-body closure. The caller supplies the transport and typed response decoder.

The root standalone client and Vine's framework adapter both use this package. `meta.Context`, `meta.Actor`, `meta.Initiator`, `MethodInfo`, schema registration, typed validation, framework errors, h2c and mTLS setup remain in Vine. The optional CBOR package is never imported by JSON-only clients.

`DecodeError` distinguishes decoder failures from network failures, allowing Vine to preserve its existing `ex.UnexpectedResponse` mapping. This package does not interpret Portal authentication or decide retries.
