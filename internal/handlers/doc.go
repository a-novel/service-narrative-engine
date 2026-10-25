// Package handlers is the transport layer of the service, sitting above the core layer. Handlers
// translate incoming requests into core calls, and map the results and sentinel errors back to
// HTTP responses and status codes. They serve the public REST API, for any HTTP client, wired by
// cmd/rest.
package handlers
