package core_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicegenai "github.com/a-novel/service-genai/pkg/go"
	servicegenaimocks "github.com/a-novel/service-genai/pkg/go/mocks"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestGenerationSubmit(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")
	validRequest := &core.GenerationSubmitRequest{
		Actor:           core.Actor{UserID: ownerID},
		IdeaID:          ideaID,
		EngineVersionID: engineVersionID,
		StepKey:         "manuscript",
		IdempotencyKey:  "retry-1",
	}
	pending := generationFixture(servicegenai.GenerationStatusPending, nil)
	succeeded := generationFixture(
		servicegenai.GenerationStatusSucceeded,
		responsesOutput(t, manuscriptValue),
	)
	constEnumEngine := engineVersionFixture()
	constEnumEngine.Definition = json.RawMessage(strings.Replace(
		string(engineDefinition),
		`"format": {"const": "prose"}`,
		`"format": {"const": "prose", "enum": ["prose", "dialogue"]}`,
		1,
	))
	conflictingEnumEngine := engineVersionFixture()
	conflictingEnumEngine.Definition = json.RawMessage(strings.Replace(
		string(engineDefinition),
		`"format": {"const": "prose"}`,
		`"format": {"const": "prose", "enum": ["dialogue"]}`,
		1,
	))

	testCases := []struct {
		name string

		request *core.GenerationSubmitRequest

		ideaResponse   *dao.Idea
		ideaErr        error
		engineResponse *dao.EngineVersion
		engineErr      error
		genaiResponse  *servicegenai.GenerationSubmitResponse
		genaiErr       error

		expect    *core.GenerationSubmitResult
		expectErr error
	}{
		{
			name:           "Success/Created",
			request:        validRequest,
			ideaResponse:   ideaFixture(),
			engineResponse: engineVersionFixture(),
			genaiResponse: &servicegenai.GenerationSubmitResponse{
				Generation: pending,
				Created:    true,
			},
			expect: &core.GenerationSubmitResult{
				Generation: expectedGeneration(core.GenerationStatusPending, nil),
				Created:    true,
			},
		},
		{
			name:           "Success/ReplaySucceeded",
			request:        validRequest,
			ideaResponse:   ideaFixture(),
			engineResponse: engineVersionFixture(),
			genaiResponse: &servicegenai.GenerationSubmitResponse{
				Generation: succeeded,
			},
			expect: &core.GenerationSubmitResult{
				Generation: expectedGeneration(core.GenerationStatusSucceeded, &manuscriptValue),
			},
		},
		{
			name:           "Success/ConstAndEnumProjection",
			request:        validRequest,
			ideaResponse:   ideaFixture(),
			engineResponse: constEnumEngine,
			genaiResponse: &servicegenai.GenerationSubmitResponse{
				Generation: pending,
				Created:    true,
			},
			expect: &core.GenerationSubmitResult{
				Generation: expectedGeneration(core.GenerationStatusPending, nil),
				Created:    true,
			},
		},
		{
			name:           "Error/ConstEnumConflict",
			request:        validRequest,
			ideaResponse:   ideaFixture(),
			engineResponse: conflictingEnumEngine,
			expectErr:      core.ErrEngineDefinitionInvalid,
		},
		{
			name:      "Error/InvalidRequest",
			request:   &core.GenerationSubmitRequest{},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name:      "Error/IdeaNotFound",
			request:   validRequest,
			ideaErr:   dao.ErrIdeaSelectNotFound,
			expectErr: core.ErrIdeaNotFound,
		},
		{
			name:      "Error/IdeaDao",
			request:   validRequest,
			ideaErr:   errFoo,
			expectErr: errFoo,
		},
		{
			name:         "Error/EngineVersionNotFound",
			request:      validRequest,
			ideaResponse: ideaFixture(),
			engineErr:    dao.ErrEngineVersionSelectNotFound,
			expectErr:    core.ErrEngineVersionNotFound,
		},
		{
			name:         "Error/EngineVersionDao",
			request:      validRequest,
			ideaResponse: ideaFixture(),
			engineErr:    errFoo,
			expectErr:    errFoo,
		},
		{
			name:         "Error/StepNotFound",
			request:      validRequest,
			ideaResponse: ideaFixture(),
			engineResponse: &dao.EngineVersion{
				ID:         engineVersionID,
				Definition: json.RawMessage(`{"steps":[]}`),
			},
			expectErr: core.ErrEngineStepNotFound,
		},
		{
			name:         "Error/DefinitionMalformed",
			request:      validRequest,
			ideaResponse: ideaFixture(),
			engineResponse: &dao.EngineVersion{
				ID:         engineVersionID,
				Definition: json.RawMessage(`{`),
			},
			expectErr: core.ErrEngineDefinitionInvalid,
		},
		{
			name:         "Error/DuplicateStep",
			request:      validRequest,
			ideaResponse: ideaFixture(),
			engineResponse: &dao.EngineVersion{
				ID: engineVersionID,
				Definition: json.RawMessage(`{"steps":[
					{"key":"manuscript","promptTemplate":"one","outputSchema":{"type":"object"}},
					{"key":"manuscript","promptTemplate":"two","outputSchema":{"type":"object"}}
				]}`),
			},
			expectErr: core.ErrEngineDefinitionInvalid,
		},
		{
			name:         "Error/IncompleteStep",
			request:      validRequest,
			ideaResponse: ideaFixture(),
			engineResponse: &dao.EngineVersion{
				ID: engineVersionID,
				Definition: json.RawMessage(
					`{"steps":[{"key":"manuscript","promptTemplate":" ","outputSchema":{"type":"object"}}]}`,
				),
			},
			expectErr: core.ErrEngineDefinitionInvalid,
		},
		{
			name:         "Error/NullProviderSchema",
			request:      validRequest,
			ideaResponse: ideaFixture(),
			engineResponse: &dao.EngineVersion{
				ID: engineVersionID,
				Definition: json.RawMessage(
					`{"steps":[{"key":"manuscript","promptTemplate":"write","outputSchema":null}]}`,
				),
			},
			expectErr: core.ErrEngineDefinitionInvalid,
		},
		{
			name:           "Error/IdempotencyConflict",
			request:        validRequest,
			ideaResponse:   ideaFixture(),
			engineResponse: engineVersionFixture(),
			genaiErr:       status.Error(codes.AlreadyExists, "conflict"),
			expectErr:      core.ErrGenerationConflict,
		},
		{
			name:           "Error/GenAI",
			request:        validRequest,
			ideaResponse:   ideaFixture(),
			engineResponse: engineVersionFixture(),
			genaiErr:       errFoo,
			expectErr:      errFoo,
		},
		{
			name:           "Error/MissingResponse",
			request:        validRequest,
			ideaResponse:   ideaFixture(),
			engineResponse: engineVersionFixture(),
			expectErr:      core.ErrGenerationResponseInvalid,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ideaDao := coremocks.NewMockIdeaSelectDao(t)
			engineVersionDao := coremocks.NewMockEngineVersionSelectDao(t)
			genai := servicegenaimocks.NewMockClient(t)

			if testCase.ideaResponse != nil || testCase.ideaErr != nil {
				ideaDao.EXPECT().
					Exec(mock.Anything, &dao.IdeaSelectRequest{
						ID:      testCase.request.IdeaID,
						OwnerID: testCase.request.Actor.UserID,
					}).
					Return(testCase.ideaResponse, testCase.ideaErr)
			}

			if testCase.engineResponse != nil || testCase.engineErr != nil {
				engineVersionDao.EXPECT().
					Exec(mock.Anything, &dao.EngineVersionSelectRequest{
						ID: testCase.request.EngineVersionID,
					}).
					Return(testCase.engineResponse, testCase.engineErr)
			}

			if testCase.genaiResponse != nil || testCase.genaiErr != nil ||
				testCase.name == "Error/MissingResponse" {
				genai.EXPECT().
					GenerationSubmit(mock.Anything, mock.MatchedBy(func(
						request *servicegenai.GenerationSubmitRequest,
					) bool {
						return assertGenerationRequest(t, request)
					})).
					Return(testCase.genaiResponse, testCase.genaiErr)
			}

			result, err := core.NewGenerationSubmit(
				ideaDao,
				engineVersionDao,
				genai,
			).Exec(t.Context(), testCase.request)

			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)
		})
	}
}

func assertGenerationRequest(
	t *testing.T,
	request *servicegenai.GenerationSubmitRequest,
) bool {
	t.Helper()

	var payload map[string]any

	err := json.Unmarshal(request.GetRequest(), &payload)
	require.NoError(t, err)

	reasoning, reasoningOK := payload["reasoning"].(map[string]any)
	text, textOK := payload["text"].(map[string]any)
	format, formatOK := text["format"].(map[string]any)
	schema, schemaOK := format["schema"].(map[string]any)
	properties, propertiesOK := schema["properties"].(map[string]any)
	valueSchema, valueSchemaOK := properties["value"].(map[string]any)
	valueProperties, valuePropertiesOK := valueSchema["properties"].(map[string]any)
	formatSchema, manuscriptFormatOK := valueProperties["format"].(map[string]any)

	// The Engine schema constrains string length; a strict Responses schema
	// rejects those keywords, so none may survive the projection at any depth.
	projected, err := json.Marshal(schema)
	require.NoError(t, err)

	return assert.Equal(t, ownerID.String(), request.GetOwnerId()) &&
		assert.Equal(t, core.GenerationPurposeStudio, request.GetPurpose()) &&
		assert.Equal(
			t,
			"213f384149d055b91505270aa05fce43951672897fe8987ee15ecec335fba707",
			request.GetIdempotencyKey(),
		) &&
		assert.Equal(t, int32(2), request.GetMaxAttempts()) &&
		assert.Equal(t, core.GenerationModelDefault, payload["model"]) &&
		assert.NotContains(t, payload, "background") &&
		assert.NotContains(t, payload, "store") &&
		assert.NotContains(t, payload, "metadata") &&
		assert.NotContains(t, payload, "previous_response_id") &&
		assert.Contains(t, payload["input"], ideaFixture().Seed) &&
		assert.True(t, reasoningOK) &&
		assert.Equal(t, "low", reasoning["effort"]) &&
		assert.True(t, textOK) &&
		assert.True(t, formatOK) &&
		assert.True(t, schemaOK) &&
		assert.Equal(t, "json_schema", format["type"]) &&
		assert.Equal(t, true, format["strict"]) &&
		assert.Equal(t, false, schema["additionalProperties"]) &&
		assert.True(t, propertiesOK) &&
		assert.True(t, valueSchemaOK) &&
		assert.NotContains(t, valueSchema, "$schema") &&
		assert.True(t, valuePropertiesOK) &&
		assert.True(t, manuscriptFormatOK) &&
		assert.NotContains(t, formatSchema, "const") &&
		assert.Equal(t, []any{"prose"}, formatSchema["enum"]) &&
		assert.NotContains(t, string(projected), "minLength") &&
		assert.NotContains(t, string(projected), "maxLength") &&
		assert.Contains(t, string(projected), "minItems")
}
