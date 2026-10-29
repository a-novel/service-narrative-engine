package schemas_test

import (
	_ "embed"
	"encoding/json"
	"testing"

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
)

// schemaCase is one document a content schema accepts or refuses.
type schemaCase struct {
	Name     string          `json:"name"`
	Valid    bool            `json:"valid"`
	Document json.RawMessage `json:"document"`
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
