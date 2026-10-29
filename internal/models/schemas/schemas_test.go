package schemas_test

import (
	_ "embed"
	"encoding/json"
	"maps"
	"slices"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/models/schemas"
)

// The cases files are shared with the JS client's tests, so both validators agree on every case.
var (
	//go:embed testdata/content.notes.cases.json
	notesCases []byte
	//go:embed testdata/content.manuscript.cases.json
	manuscriptCases []byte
	//go:embed testdata/manuscript.cases.json
	formatCases []byte
	//go:embed testdata/output.suggest.cases.json
	suggestCases []byte
	//go:embed testdata/output.consolidate.cases.json
	consolidateCases []byte
	//go:embed testdata/output.check.cases.json
	checkCases []byte
)

// schemaCase is one document a schema accepts or refuses. Strict says whether the strict form of an
// output schema accepts it.
type schemaCase struct {
	Name     string          `json:"name"`
	Strict   bool            `json:"strict"`
	Valid    bool            `json:"valid"`
	Document json.RawMessage `json:"document"`
}

// strictKeywords lists the keywords OpenAI's strict structured outputs accept.
var strictKeywords = []string{
	"$defs", "$ref", "additionalProperties", "anyOf", "description", "enum", "exclusiveMaximum",
	"exclusiveMinimum", "format", "items", "maxItems", "maximum", "minItems", "minimum", "multipleOf",
	"pattern", "properties", "required", "type",
}

// strictLimits are the sizes OpenAI's strict structured outputs accept at most.
const (
	strictMaxDepth      = 10
	strictMaxProperties = 5000
	strictMaxEnumValues = 1000
	strictMaxNamesSize  = 120000
)

// strictStats counts what the strict-mode limits cover across a document.
type strictStats struct {
	properties int
	enumValues int
	namesSize  int
}

// requireStrict fails unless schema follows strict mode's rules. Every object
// lists all its properties as required and refuses others.
func requireStrict(t *testing.T, schema map[string]any, stats *strictStats) {
	t.Helper()

	for keyword := range schema {
		require.Contains(t, strictKeywords, keyword)
	}

	if properties, ok := schema["properties"].(map[string]any); ok {
		require.Equal(t, false, schema["additionalProperties"])
		require.ElementsMatch(t, slices.Collect(maps.Keys(properties)), schema["required"])

		for name, property := range properties {
			stats.properties++
			stats.namesSize += len(name)

			requireStrict(t, property.(map[string]any), stats)
		}
	}

	if enum, ok := schema["enum"].([]any); ok {
		require.Contains(t, schema, "type", "an enum states its type")

		stats.enumValues += len(enum)
		for _, value := range enum {
			stats.namesSize += len(value.(string))
		}
	}

	if items, ok := schema["items"].(map[string]any); ok {
		requireStrict(t, items, stats)
	}

	if anyOf, ok := schema["anyOf"].([]any); ok {
		for _, branch := range anyOf {
			requireStrict(t, branch.(map[string]any), stats)
		}
	}
}

// strictDepth returns how many objects nest in schema, a part of document, following references.
func strictDepth(t *testing.T, document, schema map[string]any, seen []string) int {
	t.Helper()

	if ref, ok := schema["$ref"].(string); ok {
		require.NotContains(t, seen, ref, "a strict schema here holds no cycle")

		def := document["$defs"].(map[string]any)[ref[len("#/$defs/"):]]

		return strictDepth(t, document, def.(map[string]any), append(seen, ref))
	}

	var children []any

	if properties, ok := schema["properties"].(map[string]any); ok {
		children = slices.Collect(maps.Values(properties))
	}

	if items, ok := schema["items"]; ok {
		children = append(children, items)
	}

	if anyOf, ok := schema["anyOf"].([]any); ok {
		children = append(children, anyOf...)
	}

	var depth int

	for _, child := range children {
		depth = max(depth, strictDepth(t, document, child.(map[string]any), seen))
	}

	if schema["type"] == "object" {
		depth++
	}

	return depth
}

func TestContent(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		kind string

		expectErr error
	}{
		{
			name: "Success/Notes",
			kind: "notes",
		},
		{
			name: "Success/Manuscript",
			kind: "manuscript",
		},
		{
			name:      "Error/UnknownKind",
			kind:      "timeline",
			expectErr: schemas.ErrUnknownKind,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			schema, err := schemas.Content(testCase.kind)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expectErr == nil, schema != nil)
		})
	}
}

func TestSchema(t *testing.T) {
	t.Parallel()

	notes, err := schemas.Content("notes")
	require.NoError(t, err)

	manuscript, err := schemas.Content("manuscript")
	require.NoError(t, err)

	suites := map[string]struct {
		schema *schemas.Schema
		cases  []byte
	}{
		"notes":      {schema: notes, cases: notesCases},
		"manuscript": {schema: manuscript, cases: manuscriptCases},
		"format":     {schema: schemas.Manuscript, cases: formatCases},
	}

	for name, suite := range suites {
		var cases []schemaCase
		require.NoError(t, json.Unmarshal(suite.cases, &cases))

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			for _, testCase := range cases {
				t.Run(testCase.Name, func(t *testing.T) {
					t.Parallel()

					err := suite.schema.Validate(testCase.Document)
					if testCase.Valid {
						require.NoError(t, err)
					} else {
						require.ErrorIs(t, err, schemas.ErrInvalidDocument)
					}
				})
			}

			t.Run("Error/NotJSON", func(t *testing.T) {
				t.Parallel()

				require.ErrorIs(t, suite.schema.Validate(json.RawMessage(`{"text":`)), schemas.ErrInvalidDocument)
			})
		})
	}
}

func TestOutput(t *testing.T) {
	t.Parallel()

	suites := map[string][]byte{
		"suggest":     suggestCases,
		"consolidate": consolidateCases,
		"check":       checkCases,
	}

	t.Run("Error/UnknownKind", func(t *testing.T) {
		t.Parallel()

		_, err := schemas.Output("rewrite")
		require.ErrorIs(t, err, schemas.ErrUnknownGenerationKind)
	})

	for kind, data := range suites {
		var cases []schemaCase
		require.NoError(t, json.Unmarshal(data, &cases))

		schema, err := schemas.Output(kind)
		require.NoError(t, err)

		strictDocument, err := schema.Strict()
		require.NoError(t, err)

		strictSchema := &jsonschema.Schema{}
		require.NoError(t, json.Unmarshal(strictDocument, strictSchema))

		strict, err := strictSchema.Resolve(nil)
		require.NoError(t, err)

		t.Run(kind, func(t *testing.T) {
			t.Parallel()

			t.Run("StrictRules", func(t *testing.T) {
				t.Parallel()

				var document map[string]any
				require.NoError(t, json.Unmarshal(strictDocument, &document))

				require.Equal(t, "object", document["type"])
				require.NotContains(t, document, "anyOf")

				stats := &strictStats{}

				requireStrict(t, document, stats)

				defs, _ := document["$defs"].(map[string]any)
				for name, def := range defs {
					stats.namesSize += len(name)

					requireStrict(t, def.(map[string]any), stats)
				}

				require.LessOrEqual(t, strictDepth(t, document, document, nil), strictMaxDepth)
				require.LessOrEqual(t, stats.properties, strictMaxProperties)
				require.LessOrEqual(t, stats.enumValues, strictMaxEnumValues)
				require.LessOrEqual(t, stats.namesSize, strictMaxNamesSize)
			})

			for _, testCase := range cases {
				t.Run(testCase.Name, func(t *testing.T) {
					t.Parallel()

					var value any
					require.NoError(t, json.Unmarshal(testCase.Document, &value))

					require.Equal(t, testCase.Strict, strict.Validate(value) == nil, "strict")

					err := schema.Validate(testCase.Document)
					if testCase.Valid {
						require.NoError(t, err)
					} else {
						require.ErrorIs(t, err, schemas.ErrInvalidDocument)
					}
				})
			}
		})
	}
}
