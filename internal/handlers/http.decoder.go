package handlers

import "github.com/gorilla/schema"

// muxDecoder decodes URL query parameters into request structs tagged with `schema`.
// Shared by the handlers that read their input from the query string rather than the body.
var muxDecoder = schema.NewDecoder()
