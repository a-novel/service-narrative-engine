package services_test

import "github.com/a-novel/service-narrative-engine/internal/lib"

var testModuleSchema = &lib.RawSchema{
	Type: lib.NewSchemaType("object"),
	Properties: map[string]lib.RawSchema{
		"field1": {
			Type: lib.NewSchemaType("string"),
		},
	},
	Required: []string{"field1"},
}
