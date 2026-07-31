package core_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
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
	"github.com/a-novel/service-narrative-engine/internal/models/schemas"
)

type engineSelectCall struct {
	id       uuid.UUID
	response *dao.EngineVersion
	err      error
}

type generationPayloadExpectation struct {
	target                       core.GenerationTarget
	input                        json.RawMessage
	steps                        []*dao.StepValue
	manuscript                   json.RawMessage
	preservesApplicationKeywords bool
}

func TestGenerationSubmit(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")
	target := generationTargetFixture()
	validRequest := &core.GenerationSubmitRequest{
		Actor:  core.Actor{UserID: ownerID},
		IdeaID: ideaID,
		Target: target,
		Input:  json.RawMessage(`{"title":"A partial proposal"}`),
		ContextOverrides: []core.GenerationContextOverride{
			{
				EngineVersionID: engineVersionID,
				StepKey:         "manuscript",
				Value:           json.RawMessage(`{}`),
			},
		},
		IdempotencyKey: "retry-1",
	}
	pending := generationFixture(servicegenai.GenerationStatusPending, nil)
	historicalEngineVersionID := uuid.MustParse("00000000-0000-0000-0000-000000000099")
	savedStep := &dao.StepValue{
		ID:              uuid.MustParse("00000000-0000-0000-0000-000000000501"),
		IdeaID:          ideaID,
		EngineVersionID: historicalEngineVersionID,
		StepKey:         "characters",
		Value:           json.RawMessage(`{"names":["Mara"]}`),
		CreatedAt:       createdAt,
	}
	latestManuscript := json.RawMessage(
		`{"blocks":[{"type":"text","metadata":{"source":"saved"},` +
			`"data":{"text":"The buried foghorn answers.","marks":[]}}]}`,
	)
	manuscript := &dao.Manuscript{
		ID:        uuid.MustParse("00000000-0000-0000-0000-000000000502"),
		IdeaID:    ideaID,
		Value:     latestManuscript,
		CreatedAt: createdAt,
	}
	ideaTarget := core.GenerationTarget{Kind: core.GenerationTargetKindIdea}
	ideaProposal := json.RawMessage(
		`{"title":"The Answering Light","genre":"speculative",` +
			`"seed":"A second foghorn answers from below."}`,
	)
	ideaRequest := &core.GenerationSubmitRequest{
		Actor:          validRequest.Actor,
		IdeaID:         ideaID,
		Target:         ideaTarget,
		Input:          json.RawMessage(`{"title":"The Answering Light"}`),
		IdempotencyKey: "retry-idea",
	}
	manuscriptTarget := core.GenerationTarget{Kind: core.GenerationTargetKindManuscript}
	manuscriptRequest := &core.GenerationSubmitRequest{
		Actor:          validRequest.Actor,
		IdeaID:         ideaID,
		Target:         manuscriptTarget,
		Input:          json.RawMessage(`{"blocks":[{"type":"text","metadata":{},"data":{"text":"Draft"}}]}`),
		IdempotencyKey: "retry-manuscript",
	}
	providerProjectionTarget := core.GenerationTarget{
		Kind:            core.GenerationTargetKindStep,
		EngineVersionID: engineVersionID,
		StepKey:         "provider-projection",
	}
	providerProjectionRequest := &core.GenerationSubmitRequest{
		Actor:          validRequest.Actor,
		IdeaID:         ideaID,
		Target:         providerProjectionTarget,
		Input:          json.RawMessage(`{"minLength":"draft"}`),
		IdempotencyKey: "retry-provider-projection",
	}
	duplicateOverrides := *validRequest
	duplicateOverrides.ContextOverrides = append(
		append([]core.GenerationContextOverride{}, validRequest.ContextOverrides...),
		validRequest.ContextOverrides[0],
	)
	invalidInput := *validRequest
	invalidInput.Input = json.RawMessage(`{"unknown":true}`)
	overrideAtLimit := *validRequest
	overrideAtLimit.ContextOverrides = []core.GenerationContextOverride{{
		EngineVersionID: engineVersionID,
		StepKey:         "document",
		Value:           contentDocumentOfSize(schemas.ContentDocumentMaxBytes),
	}}
	overrideOverLimit := overrideAtLimit
	overrideOverLimit.ContextOverrides = []core.GenerationContextOverride{{
		EngineVersionID: engineVersionID,
		StepKey:         "document",
		Value:           contentDocumentOfSize(schemas.ContentDocumentMaxBytes + 1),
	}}
	stepSelectFailure := *validRequest
	stepSelectFailure.ContextOverrides = nil
	manuscriptSelectFailure := stepSelectFailure
	genaiFailure := stepSelectFailure

	testCases := []struct {
		name string

		request *core.GenerationSubmitRequest

		accessResponse *dao.Idea
		accessErr      error
		callAccess     bool
		engineCalls    []engineSelectCall
		stepResponse   []*dao.StepValue
		stepErr        error
		callStep       bool
		manuscriptResp *dao.Manuscript
		manuscriptErr  error
		callManuscript bool
		genaiResponse  *servicegenai.GenerationSubmitResponse
		genaiErr       error
		callGenAI      bool

		payload *generationPayloadExpectation
		expect  *core.GenerationSubmitResult
		err     error
	}{
		{
			name:           "Success/LatestContextAndOverride",
			request:        validRequest,
			accessResponse: ideaFixture(),
			callAccess:     true,
			engineCalls: []engineSelectCall{
				{id: engineVersionID, response: engineVersionFixture()},
				{id: engineVersionID, response: engineVersionFixture()},
			},
			stepResponse:   []*dao.StepValue{savedStep},
			callStep:       true,
			manuscriptResp: manuscript,
			callManuscript: true,
			genaiResponse: &servicegenai.GenerationSubmitResponse{
				Generation: pending,
				Created:    true,
			},
			callGenAI: true,
			payload: &generationPayloadExpectation{
				target: target,
				input:  validRequest.Input,
				steps: []*dao.StepValue{
					savedStep,
					{
						EngineVersionID: engineVersionID,
						StepKey:         "manuscript",
						Value:           json.RawMessage(`{}`),
					},
				},
				manuscript: latestManuscript,
			},
			expect: &core.GenerationSubmitResult{
				Generation: submittedGeneration(core.GenerationStatusPending, nil, target),
				Created:    true,
			},
		},
		{
			name:           "Success/ContextOverrideAtDocumentLimit",
			request:        &overrideAtLimit,
			accessResponse: ideaFixture(),
			callAccess:     true,
			engineCalls: []engineSelectCall{
				{id: engineVersionID, response: engineVersionFixture()},
				{id: engineVersionID, response: validationEngineVersionFixture()},
			},
			stepResponse:   []*dao.StepValue{},
			callStep:       true,
			manuscriptErr:  dao.ErrManuscriptSelectLatestNotFound,
			callManuscript: true,
			genaiResponse: &servicegenai.GenerationSubmitResponse{
				Generation: pending,
				Created:    true,
			},
			callGenAI: true,
			payload: &generationPayloadExpectation{
				target: target,
				input:  overrideAtLimit.Input,
				steps: []*dao.StepValue{{
					EngineVersionID: engineVersionID,
					StepKey:         "document",
					Value:           overrideAtLimit.ContextOverrides[0].Value,
				}},
			},
			expect: &core.GenerationSubmitResult{
				Generation: submittedGeneration(core.GenerationStatusPending, nil, target),
				Created:    true,
			},
		},
		{
			name:           "Error/ContextOverrideOverDocumentLimit",
			request:        &overrideOverLimit,
			accessResponse: ideaFixture(),
			callAccess:     true,
			engineCalls: []engineSelectCall{
				{id: engineVersionID, response: engineVersionFixture()},
				{id: engineVersionID, response: validationEngineVersionFixture()},
			},
			err: core.ErrInvalidRequest,
		},
		{
			name:           "Success/StaticIdeaWithoutPreIdeaGeneration",
			request:        ideaRequest,
			accessResponse: ideaFixture(),
			callAccess:     true,
			stepResponse:   []*dao.StepValue{},
			callStep:       true,
			manuscriptErr:  dao.ErrManuscriptSelectLatestNotFound,
			callManuscript: true,
			genaiResponse: &servicegenai.GenerationSubmitResponse{
				Generation: generationFixture(
					servicegenai.GenerationStatusSucceeded,
					responsesOutputText(
						t,
						generationEnvelopeForTarget(t, ideaTarget, ideaProposal),
					),
				),
			},
			callGenAI: true,
			payload: &generationPayloadExpectation{
				target: ideaTarget,
				input:  ideaRequest.Input,
				steps:  []*dao.StepValue{},
			},
			expect: &core.GenerationSubmitResult{
				Generation: submittedGeneration(
					core.GenerationStatusSucceeded,
					ideaProposal,
					ideaTarget,
				),
			},
		},
		{
			name:           "Success/StaticManuscriptProviderProjection",
			request:        manuscriptRequest,
			accessResponse: ideaFixture(),
			callAccess:     true,
			stepResponse:   []*dao.StepValue{},
			callStep:       true,
			manuscriptResp: manuscript,
			callManuscript: true,
			genaiResponse: &servicegenai.GenerationSubmitResponse{
				Generation: generationFixture(servicegenai.GenerationStatusPending, nil),
				Created:    true,
			},
			callGenAI: true,
			payload: &generationPayloadExpectation{
				target:     manuscriptTarget,
				input:      manuscriptRequest.Input,
				steps:      []*dao.StepValue{},
				manuscript: latestManuscript,
			},
			expect: &core.GenerationSubmitResult{
				Generation: submittedGeneration(
					core.GenerationStatusPending,
					nil,
					manuscriptTarget,
				),
				Created: true,
			},
		},
		{
			name:           "Success/ProviderProjectionPreservesApplicationKeywords",
			request:        providerProjectionRequest,
			accessResponse: ideaFixture(),
			callAccess:     true,
			engineCalls: []engineSelectCall{
				{id: engineVersionID, response: validationEngineVersionFixture()},
			},
			stepResponse:   []*dao.StepValue{},
			callStep:       true,
			manuscriptErr:  dao.ErrManuscriptSelectLatestNotFound,
			callManuscript: true,
			genaiResponse: &servicegenai.GenerationSubmitResponse{
				Generation: generationFixture(servicegenai.GenerationStatusPending, nil),
				Created:    true,
			},
			callGenAI: true,
			payload: &generationPayloadExpectation{
				target:                       providerProjectionTarget,
				input:                        providerProjectionRequest.Input,
				steps:                        []*dao.StepValue{},
				preservesApplicationKeywords: true,
			},
			expect: &core.GenerationSubmitResult{
				Generation: submittedGeneration(
					core.GenerationStatusPending,
					nil,
					providerProjectionTarget,
				),
				Created: true,
			},
		},
		{
			name:    "Error/InvalidRequest",
			request: &core.GenerationSubmitRequest{},
			err:     core.ErrInvalidRequest,
		},
		{
			name:       "Error/ProjectAccess",
			request:    validRequest,
			accessErr:  core.ErrIdeaNotFound,
			callAccess: true,
			err:        core.ErrIdeaNotFound,
		},
		{
			name:           "Error/EngineVersionNotFound",
			request:        validRequest,
			accessResponse: ideaFixture(),
			callAccess:     true,
			engineCalls: []engineSelectCall{
				{id: engineVersionID, err: dao.ErrEngineVersionSelectNotFound},
			},
			err: core.ErrEngineVersionNotFound,
		},
		{
			name:           "Error/PartialInputShape",
			request:        &invalidInput,
			accessResponse: ideaFixture(),
			callAccess:     true,
			engineCalls: []engineSelectCall{
				{id: engineVersionID, response: engineVersionFixture()},
			},
			err: core.ErrInvalidRequest,
		},
		{
			name:           "Error/DuplicateOverride",
			request:        &duplicateOverrides,
			accessResponse: ideaFixture(),
			callAccess:     true,
			engineCalls: []engineSelectCall{
				{id: engineVersionID, response: engineVersionFixture()},
				{id: engineVersionID, response: engineVersionFixture()},
			},
			err: core.ErrInvalidRequest,
		},
		{
			name:           "Error/StepContextDao",
			request:        &stepSelectFailure,
			accessResponse: ideaFixture(),
			callAccess:     true,
			engineCalls: []engineSelectCall{
				{id: engineVersionID, response: engineVersionFixture()},
			},
			stepErr:  errFoo,
			callStep: true,
			err:      errFoo,
		},
		{
			name:           "Error/ManuscriptContextDao",
			request:        &manuscriptSelectFailure,
			accessResponse: ideaFixture(),
			callAccess:     true,
			engineCalls: []engineSelectCall{
				{id: engineVersionID, response: engineVersionFixture()},
			},
			stepResponse:   []*dao.StepValue{},
			callStep:       true,
			manuscriptErr:  errFoo,
			callManuscript: true,
			err:            errFoo,
		},
		{
			name:           "Error/GenAI",
			request:        &genaiFailure,
			accessResponse: ideaFixture(),
			callAccess:     true,
			engineCalls: []engineSelectCall{
				{id: engineVersionID, response: engineVersionFixture()},
			},
			stepResponse:   []*dao.StepValue{},
			callStep:       true,
			manuscriptErr:  dao.ErrManuscriptSelectLatestNotFound,
			callManuscript: true,
			genaiErr:       status.Error(codes.AlreadyExists, "conflict"),
			callGenAI:      true,
			err:            core.ErrGenerationConflict,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			projectAccess := coremocks.NewMockProjectAccessService(t)
			engineVersionDao := coremocks.NewMockEngineVersionSelectDao(t)
			stepValueDao := coremocks.NewMockStepValueSelectLatestDao(t)
			manuscriptDao := coremocks.NewMockManuscriptSelectLatestDao(t)
			genai := servicegenaimocks.NewMockClient(t)

			if testCase.callAccess {
				projectAccess.EXPECT().
					Exec(mock.Anything, &core.ProjectAccessRequest{
						Actor:  testCase.request.Actor,
						IdeaID: testCase.request.IdeaID,
					}).
					Return(testCase.accessResponse, testCase.accessErr)
			}

			for _, call := range testCase.engineCalls {
				engineVersionDao.EXPECT().
					Exec(mock.Anything, &dao.EngineVersionSelectRequest{ID: call.id}).
					Return(call.response, call.err).
					Once()
			}

			if testCase.callStep {
				excludedStepKeys := make([]string, 0, len(testCase.request.ContextOverrides))
				for _, override := range testCase.request.ContextOverrides {
					excludedStepKeys = append(excludedStepKeys, override.StepKey)
				}

				stepValueDao.EXPECT().
					Exec(mock.Anything, &dao.StepValueSelectLatestRequest{
						IdeaID:          testCase.request.IdeaID,
						ExcludeStepKeys: excludedStepKeys,
					}).
					Return(testCase.stepResponse, testCase.stepErr)
			}

			if testCase.callManuscript {
				manuscriptDao.EXPECT().
					Exec(mock.Anything, &dao.ManuscriptSelectLatestRequest{
						IdeaID: testCase.request.IdeaID,
					}).
					Return(testCase.manuscriptResp, testCase.manuscriptErr)
			}

			if testCase.callGenAI {
				genai.EXPECT().
					GenerationSubmit(mock.Anything, mock.MatchedBy(func(
						request *servicegenai.GenerationSubmitRequest,
					) bool {
						if testCase.payload == nil {
							return true
						}

						return assertGenerationRequest(t, request, testCase.payload)
					})).
					Return(testCase.genaiResponse, testCase.genaiErr)
			}

			result, err := core.NewGenerationSubmit(
				projectAccess,
				engineVersionDao,
				stepValueDao,
				manuscriptDao,
				genai,
			).Exec(t.Context(), testCase.request)

			require.ErrorIs(t, err, testCase.err)
			require.Equal(t, testCase.expect, result)
		})
	}
}

func submittedGeneration(
	status core.GenerationStatus,
	proposal json.RawMessage,
	target core.GenerationTarget,
) *core.Generation {
	generation := expectedGeneration(status, proposal)
	generation.Target = &target

	return generation
}

func assertGenerationRequest(
	t *testing.T,
	request *servicegenai.GenerationSubmitRequest,
	expect *generationPayloadExpectation,
) bool {
	t.Helper()

	var payload struct {
		Model     string `json:"model"`
		Reasoning struct {
			Effort string `json:"effort"`
		} `json:"reasoning"`
		Instructions string `json:"instructions"`
		Input        string `json:"input"`
		Text         struct {
			Format struct {
				Type   string         `json:"type"`
				Name   string         `json:"name"`
				Schema map[string]any `json:"schema"`
				Strict bool           `json:"strict"`
			} `json:"format"`
		} `json:"text"`
	}

	err := json.Unmarshal(request.GetRequest(), &payload)
	require.NoError(t, err)

	separator := strings.IndexByte(payload.Input, '\n')
	require.NotEqual(t, -1, separator)

	var inputDocument struct {
		Target         core.GenerationTarget `json:"target"`
		TargetInput    json.RawMessage       `json:"targetInput"`
		ProjectContext struct {
			Idea struct {
				ID    uuid.UUID `json:"id"`
				Seed  string    `json:"seed"`
				Genre string    `json:"genre"`
				Title string    `json:"title"`
			} `json:"idea"`
			Steps []struct {
				EngineVersionID uuid.UUID       `json:"engineVersionID"`
				StepKey         string          `json:"stepKey"`
				Value           json.RawMessage `json:"value"`
			} `json:"steps"`
			Manuscript json.RawMessage `json:"manuscript"`
		} `json:"projectContext"`
	}

	err = json.Unmarshal([]byte(payload.Input[separator+1:]), &inputDocument)
	require.NoError(t, err)

	require.Equal(t, expect.target, inputDocument.Target)
	require.JSONEq(t, string(expect.input), string(inputDocument.TargetInput))
	require.Equal(t, ideaFixture().ID, inputDocument.ProjectContext.Idea.ID)
	require.Equal(t, ideaFixture().Seed, inputDocument.ProjectContext.Idea.Seed)
	require.Equal(t, ideaFixture().Genre, inputDocument.ProjectContext.Idea.Genre)
	require.Equal(t, ideaFixture().Title, inputDocument.ProjectContext.Idea.Title)
	require.Len(t, inputDocument.ProjectContext.Steps, len(expect.steps))

	for index, expectedStep := range expect.steps {
		actualStep := inputDocument.ProjectContext.Steps[index]
		require.Equal(t, expectedStep.EngineVersionID, actualStep.EngineVersionID)
		require.Equal(t, expectedStep.StepKey, actualStep.StepKey)
		require.JSONEq(t, string(expectedStep.Value), string(actualStep.Value))
	}

	if expect.manuscript == nil {
		require.Empty(t, inputDocument.ProjectContext.Manuscript)
	} else {
		require.JSONEq(t, string(expect.manuscript), string(inputDocument.ProjectContext.Manuscript))
	}

	schema := payload.Text.Format.Schema
	properties, propertiesOK := schema["properties"].(map[string]any)
	require.True(t, propertiesOK)

	targetKind, targetKindOK := properties["targetKind"].(map[string]any)
	engineVersion, engineVersionOK := properties["engineVersionID"].(map[string]any)
	stepKey, stepKeyOK := properties["stepKey"].(map[string]any)
	valueSchema, valueSchemaOK := properties["value"].(map[string]any)

	require.True(t, targetKindOK)
	require.True(t, engineVersionOK)
	require.True(t, stepKeyOK)
	require.True(t, valueSchemaOK)

	expectedEngineVersionID := ""
	expectedStepKey := ""

	if expect.target.Kind == core.GenerationTargetKindStep {
		expectedEngineVersionID = expect.target.EngineVersionID.String()
		expectedStepKey = expect.target.StepKey
	}

	if expect.target.Kind == core.GenerationTargetKindManuscript {
		assertManuscriptProviderSchema(t, valueSchema)
	}

	projectionValid := true

	if expect.preservesApplicationKeywords {
		assertProviderProjectionPreservesApplicationKeywords(t, valueSchema)
	} else {
		projected, marshalErr := json.Marshal(schema)
		require.NoError(t, marshalErr)

		projectionValid = assert.NotContains(t, valueSchema, "$schema") &&
			assert.NotContains(t, string(projected), "minLength") &&
			assert.NotContains(t, string(projected), "maxLength")
	}

	return assert.Equal(t, ownerID.String(), request.GetOwnerId()) &&
		assert.Equal(t, core.GenerationPurposeStudio, request.GetPurpose()) &&
		assert.Len(t, request.GetIdempotencyKey(), 64) &&
		assert.NotEqual(t, "retry-1", request.GetIdempotencyKey()) &&
		assert.Equal(t, int32(2), request.GetMaxAttempts()) &&
		assert.Equal(t, core.GenerationModelDefault, payload.Model) &&
		assert.Equal(t, core.GenerationReasoningEffortDefault, payload.Reasoning.Effort) &&
		assert.NotEmpty(t, payload.Instructions) &&
		assert.Equal(t, "json_schema", payload.Text.Format.Type) &&
		assert.Equal(t, "project_content_output", payload.Text.Format.Name) &&
		assert.True(t, payload.Text.Format.Strict) &&
		assert.Equal(t, false, schema["additionalProperties"]) &&
		assert.Equal(t, []any{string(expect.target.Kind)}, targetKind["enum"]) &&
		assert.Equal(t, []any{expectedEngineVersionID}, engineVersion["enum"]) &&
		assert.Equal(t, []any{expectedStepKey}, stepKey["enum"]) &&
		projectionValid
}

func assertProviderProjectionPreservesApplicationKeywords(t *testing.T, schema map[string]any) {
	t.Helper()

	require.NotContains(t, schema, "$schema")

	properties, propertiesOK := schema["properties"].(map[string]any)
	require.True(t, propertiesOK)
	require.Contains(t, properties, "minLength")
	require.Contains(t, properties, "$schema")

	minLengthProperty, minLengthPropertyOK := properties["minLength"].(map[string]any)
	require.True(t, minLengthPropertyOK)
	require.NotContains(t, minLengthProperty, "minLength")

	settings, settingsOK := properties["settings"].(map[string]any)
	require.True(t, settingsOK)
	require.NotContains(t, settings, "const")

	enum, enumOK := settings["enum"].([]any)
	require.True(t, enumOK)
	require.Len(t, enum, 1)
	require.Equal(t, map[string]any{
		"minLength": "kept",
		"maxLength": "kept",
		"$schema":   "kept",
		"const":     "kept",
	}, enum[0])
}

func assertManuscriptProviderSchema(t *testing.T, schema map[string]any) {
	t.Helper()

	properties, propertiesOK := schema["properties"].(map[string]any)
	require.True(t, propertiesOK)

	blocks, blocksOK := properties["blocks"].(map[string]any)
	require.True(t, blocksOK)

	block, blockOK := blocks["items"].(map[string]any)
	require.True(t, blockOK)

	blockProperties, blockPropertiesOK := block["properties"].(map[string]any)
	require.True(t, blockPropertiesOK)

	metadata, metadataOK := blockProperties["metadata"].(map[string]any)
	require.True(t, metadataOK)

	data, dataOK := blockProperties["data"].(map[string]any)
	require.True(t, dataOK)

	dataProperties, dataPropertiesOK := data["properties"].(map[string]any)
	require.True(t, dataPropertiesOK)

	require.ElementsMatch(t, []any{"type", "metadata", "data"}, block["required"])
	require.Equal(t, "object", metadata["type"])
	require.Equal(t, false, metadata["additionalProperties"])
	require.Contains(t, metadata, "properties")
	require.Empty(t, metadata["properties"])
	require.Contains(t, metadata, "required")
	require.Empty(t, metadata["required"])
	require.Equal(t, false, data["additionalProperties"])
	require.ElementsMatch(t, []any{"text", "marks"}, data["required"])
	require.Contains(t, dataProperties, "text")
	require.Contains(t, dataProperties, "marks")
}
