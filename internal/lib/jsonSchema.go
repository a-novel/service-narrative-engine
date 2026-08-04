package lib

const (
	jsonSchemaKeywordAdditionalProperties = "additionalProperties"
	jsonSchemaKeywordAnyOf                = "anyOf"
	jsonSchemaKeywordDefs                 = "$defs"
	jsonSchemaKeywordProperties           = "properties"
	jsonSchemaKeywordRequired             = "required"
)

var jsonSchemaSingleSubschemaKeywords = [...]string{
	"additionalItems",
	jsonSchemaKeywordAdditionalProperties,
	"contains",
	"contentSchema",
	"else",
	"items",
	"then",
	"unevaluatedItems",
	"unevaluatedProperties",
}

var jsonSchemaPredicateSubschemaKeywords = [...]string{
	"if",
	"not",
	"propertyNames",
}

var jsonSchemaSubschemaArrayKeywords = [...]string{
	"allOf",
	jsonSchemaKeywordAnyOf,
	"oneOf",
	"prefixItems",
}

var jsonSchemaSubschemaMapKeywords = [...]string{
	jsonSchemaKeywordDefs,
	"definitions",
	"dependencies",
	"dependentSchemas",
	"patternProperties",
	jsonSchemaKeywordProperties,
}

// WalkJSONSchema visits only schema-bearing keyword values. includePredicates
// controls traversal into schemas whose required fields define a condition or
// prohibition rather than content presence.
func WalkJSONSchema(
	value any,
	includePredicates bool,
	visit func(map[string]any) error,
) error {
	schema, ok := value.(map[string]any)
	if !ok {
		return nil
	}

	err := visit(schema)
	if err != nil {
		return err
	}

	for _, keyword := range jsonSchemaSingleSubschemaKeywords {
		err = walkJSONSchemaValue(schema[keyword], includePredicates, visit)
		if err != nil {
			return err
		}
	}

	if includePredicates {
		for _, keyword := range jsonSchemaPredicateSubschemaKeywords {
			err = walkJSONSchemaValue(schema[keyword], true, visit)
			if err != nil {
				return err
			}
		}
	}

	for _, keyword := range jsonSchemaSubschemaArrayKeywords {
		children, childrenOK := schema[keyword].([]any)
		if !childrenOK {
			continue
		}

		for _, child := range children {
			err = walkJSONSchemaValue(child, includePredicates, visit)
			if err != nil {
				return err
			}
		}
	}

	for _, keyword := range jsonSchemaSubschemaMapKeywords {
		children, childrenOK := schema[keyword].(map[string]any)
		if !childrenOK {
			continue
		}

		for _, child := range children {
			err = walkJSONSchemaValue(child, includePredicates, visit)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// walkJSONSchemaValue descends through either a single schema or a
// tuple-encoded schema list while ignoring application data.
func walkJSONSchemaValue(
	value any,
	includePredicates bool,
	visit func(map[string]any) error,
) error {
	switch value := value.(type) {
	case map[string]any:
		return WalkJSONSchema(value, includePredicates, visit)
	case []any:
		for _, child := range value {
			err := WalkJSONSchema(child, includePredicates, visit)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
