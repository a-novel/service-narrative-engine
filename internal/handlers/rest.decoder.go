package handlers

import (
	"reflect"

	"github.com/google/uuid"
	"github.com/gorilla/schema"
)

// muxDecoder decodes URL query parameters into the request structs of the endpoints that read their
// input from the query, following their `schema` tags. A repeated parameter fills a slice.
var muxDecoder = newMuxDecoder()

// newMuxDecoder registers the UUID converter, which gorilla/schema needs to fill a slice of UUIDs. A
// malformed UUID yields an invalid value, so the decode fails.
func newMuxDecoder() *schema.Decoder {
	decoder := schema.NewDecoder()
	decoder.RegisterConverter(uuid.UUID{}, func(value string) reflect.Value {
		id, err := uuid.Parse(value)
		if err != nil {
			return reflect.Value{}
		}

		return reflect.ValueOf(id)
	})

	return decoder
}
