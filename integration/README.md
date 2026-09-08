# Portal conformance tests

This is a test-only module. Its `go.yorun.ai/vine/compat/vrpc-go` module path deliberately permits imports of Vine internal packages for compatibility testing; it is not a library or a published module. Production code never imports Vine.

Tests pin Vine v0.15.3 and use its actual RpcGW, access-control, metadata parsers, and response writer. A local HTTP listener hosts Portal. An in-process ingress fixture supplies application/auth responses; Vine's in-memory Redis fixture supplies schema/discovery data. No deployed Hub, Link, Redis, or backend mTLS is exercised.

The relative `replace go.yorun.ai/vrpc => ..` always tests the current checkout. Do not replace Vine with an uncommitted local checkout for the release compatibility gate.

```sh
GOWORK=off go test -race ./...
GOWORK=off go vet ./...
```
