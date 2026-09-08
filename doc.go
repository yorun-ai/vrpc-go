// Package vrpc implements a standalone HTTP client for the vRPC protocol.
// It can call Vine Portal without starting a Vine application. Endpoint URLs
// include the invocation prefix (for example https://api.example.com/invoke).
// Clients are safe for concurrent calls when their transport and authorization
// callback are also safe for concurrent use. Calls are never retried by this package.
package vrpc
