// Package vrpc implements a standalone HTTP client for the vRPC protocol.
// It calls a Vine Portal endpoint without starting a Vine application. An
// endpoint URL is the invocation prefix, for example
// https://api.example.com/invoke. A client is safe for concurrent use when its
// transport is also concurrency-safe. The package does not retry calls.
package vrpc
