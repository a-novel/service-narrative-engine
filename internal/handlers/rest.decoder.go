package handlers

import "github.com/gorilla/schema"

// muxDecoder decodes URL query parameters into the request structs of the endpoints that read their
// input from the query, following their `schema` tags.
var muxDecoder = schema.NewDecoder()
