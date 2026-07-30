package core_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
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

func TestGenerationGet(t *testing.T) {
	t.Parallel()

	const privateGenerationOutput = "do-not-trace-generation-output"

	errFoo := errors.New("foo")
	validRequest := &core.GenerationGetRequest{
		Actor: core.Actor{UserID: ownerID},
		ID:    generationID,
	}
	failed := generationFixture(servicegenai.GenerationStatusFailed, nil)
	failed.Error = "provider failed"
	expectFailed := expectedGeneration(core.GenerationStatusFailed, nil)
	expectFailed.Failure = "provider failed"
	invalidValue := map[string]any{
		"title":  "The Answering Light",
		"format": privateGenerationOutput,
		"scenes": []any{map[string]any{
			"title": "The Reply",
			"blocks": []any{map[string]any{
				"kind": "prose",
				"text": "The buried foghorn answers.",
			}},
		}},
	}
	mismatchedOwner := generationFixture(servicegenai.GenerationStatusPending, nil)
	mismatchedOwner.OwnerId = uuid.MustParse("00000000-0000-0000-0000-000000000043").String()
	mismatchedPurpose := generationFixture(servicegenai.GenerationStatusPending, nil)
	mismatchedPurpose.Purpose = "other"
	mismatchedID := generationFixture(servicegenai.GenerationStatusPending, nil)
	mismatchedID.Id = uuid.MustParse("00000000-0000-0000-0000-000000000699").String()
	invalidTime := generationFixture(servicegenai.GenerationStatusPending, nil)
	invalidTime.UpdatedAt = "not-a-time"
	invalidGenerationID := generationFixture(servicegenai.GenerationStatusPending, nil)
	invalidGenerationID.Id = "not-a-uuid"
	invalidOwnerID := generationFixture(servicegenai.GenerationStatusPending, nil)
	invalidOwnerID.OwnerId = "not-a-uuid"
	missingCreatedAt := generationFixture(servicegenai.GenerationStatusPending, nil)
	missingCreatedAt.CreatedAt = ""
	invalidSettledAt := generationFixture(servicegenai.GenerationStatusFailed, nil)
	invalidSettledAt.SettledAt = "not-a-time"
	missingStepEngine := engineVersionFixture()
	missingStepEngine.Definition = json.RawMessage(`{"steps":[]}`)
	schemaDefinedValue := json.RawMessage(`{"characters":["Mara"]}`)
	schemaDefinedEngine := validationEngineVersionFixture()
	manuscriptTarget := core.GenerationTarget{Kind: core.GenerationTargetKindManuscript}

	testCases := []struct {
		name string

		request        *core.GenerationGetRequest
		genaiResponse  *servicegenai.GenerationGetResponse
		genaiErr       error
		callGenAI      bool
		engineResponse *dao.EngineVersion
		engineErr      error

		expect            *core.Generation
		expectErr         error
		expectErrExcludes string
	}{
		{
			name:      "Success/Pending",
			request:   validRequest,
			callGenAI: true,
			genaiResponse: &servicegenai.GenerationGetResponse{
				Generation: generationFixture(servicegenai.GenerationStatusPending, nil),
			},
			expect: expectedGeneration(core.GenerationStatusPending, nil),
		},
		{
			name:      "Success/Running",
			request:   validRequest,
			callGenAI: true,
			genaiResponse: &servicegenai.GenerationGetResponse{
				Generation: generationFixture(servicegenai.GenerationStatusRunning, nil),
			},
			expect: expectedGeneration(core.GenerationStatusRunning, nil),
		},
		{
			name:      "Success/Succeeded/NestedOutput",
			request:   validRequest,
			callGenAI: true,
			genaiResponse: &servicegenai.GenerationGetResponse{
				Generation: generationFixture(
					servicegenai.GenerationStatusSucceeded,
					responsesOutput(t, manuscriptValue),
				),
			},
			engineResponse: engineVersionFixture(),
			expect:         expectedGeneration(core.GenerationStatusSucceeded, manuscriptValue),
		},
		{
			name:      "Success/Succeeded/TopLevelOutput",
			request:   validRequest,
			callGenAI: true,
			genaiResponse: &servicegenai.GenerationGetResponse{
				Generation: generationFixture(
					servicegenai.GenerationStatusSucceeded,
					responsesTopLevelOutput(t, manuscriptValue),
				),
			},
			engineResponse: engineVersionFixture(),
			expect:         expectedGeneration(core.GenerationStatusSucceeded, manuscriptValue),
		},
		{
			name:      "Success/Succeeded/SchemaDefinedProposal",
			request:   validRequest,
			callGenAI: true,
			genaiResponse: &servicegenai.GenerationGetResponse{
				Generation: generationFixture(
					servicegenai.GenerationStatusSucceeded,
					responsesOutput(t, schemaDefinedValue),
				),
			},
			engineResponse: schemaDefinedEngine,
			expect:         expectedGeneration(core.GenerationStatusSucceeded, schemaDefinedValue),
		},
		{
			name:      "Success/Succeeded/StaticManuscript",
			request:   validRequest,
			callGenAI: true,
			genaiResponse: &servicegenai.GenerationGetResponse{
				Generation: generationFixture(
					servicegenai.GenerationStatusSucceeded,
					responsesOutputText(
						t,
						generationEnvelopeForTarget(
							t,
							manuscriptTarget,
							staticManuscriptValue,
						),
					),
				),
			},
			expect: submittedGeneration(
				core.GenerationStatusSucceeded,
				staticManuscriptValue,
				manuscriptTarget,
			),
		},
		{
			name:          "Success/Failed",
			request:       validRequest,
			callGenAI:     true,
			genaiResponse: &servicegenai.GenerationGetResponse{Generation: failed},
			expect:        expectFailed,
		},
		{
			name:      "Success/Abandoned",
			request:   validRequest,
			callGenAI: true,
			genaiResponse: &servicegenai.GenerationGetResponse{
				Generation: generationFixture(servicegenai.GenerationStatusAbandoned, nil),
			},
			expect: expectedGeneration(core.GenerationStatusAbandoned, nil),
		},
		{
			name:      "Success/Cancelled",
			request:   validRequest,
			callGenAI: true,
			genaiResponse: &servicegenai.GenerationGetResponse{
				Generation: generationFixture(servicegenai.GenerationStatusCancelled, nil),
			},
			expect: expectedGeneration(core.GenerationStatusCancelled, nil),
		},
		{
			name:      "Error/InvalidRequest",
			request:   &core.GenerationGetRequest{},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name:      "Error/NotFound",
			request:   validRequest,
			callGenAI: true,
			genaiErr:  status.Error(codes.NotFound, "not found"),
			expectErr: core.ErrGenerationNotFound,
		},
		{
			name:      "Error/GenAI",
			request:   validRequest,
			callGenAI: true,
			genaiErr:  errFoo,
			expectErr: errFoo,
		},
		{
			name:      "Error/MissingResponse",
			request:   validRequest,
			callGenAI: true,
			expectErr: core.ErrGenerationResponseInvalid,
		},
		{
			name:          "Error/MissingGeneration",
			request:       validRequest,
			callGenAI:     true,
			genaiResponse: &servicegenai.GenerationGetResponse{},
			expectErr:     core.ErrGenerationResponseInvalid,
		},
		{
			name:      "Error/InvalidValue",
			request:   validRequest,
			callGenAI: true,
			genaiResponse: &servicegenai.GenerationGetResponse{
				Generation: generationFixture(
					servicegenai.GenerationStatusSucceeded,
					responsesOutput(t, invalidValue),
				),
			},
			engineResponse:    engineVersionFixture(),
			expectErr:         core.ErrGenerationOutputInvalid,
			expectErrExcludes: privateGenerationOutput,
		},
		{
			name:      "Error/EngineVersionNotFound",
			request:   validRequest,
			callGenAI: true,
			genaiResponse: &servicegenai.GenerationGetResponse{
				Generation: generationFixture(
					servicegenai.GenerationStatusSucceeded,
					responsesOutput(t, manuscriptValue),
				),
			},
			engineErr: dao.ErrEngineVersionSelectNotFound,
			expectErr: core.ErrEngineVersionNotFound,
		},
		{
			name:      "Error/EngineVersionDao",
			request:   validRequest,
			callGenAI: true,
			genaiResponse: &servicegenai.GenerationGetResponse{
				Generation: generationFixture(
					servicegenai.GenerationStatusSucceeded,
					responsesOutput(t, manuscriptValue),
				),
			},
			engineErr: errFoo,
			expectErr: errFoo,
		},
		{
			name:      "Error/EngineStepMissing",
			request:   validRequest,
			callGenAI: true,
			genaiResponse: &servicegenai.GenerationGetResponse{
				Generation: generationFixture(
					servicegenai.GenerationStatusSucceeded,
					responsesOutput(t, manuscriptValue),
				),
			},
			engineResponse: missingStepEngine,
			expectErr:      core.ErrGenerationOutputInvalid,
		},
		{
			name:      "Error/EmptyOutput",
			request:   validRequest,
			callGenAI: true,
			genaiResponse: &servicegenai.GenerationGetResponse{
				Generation: generationFixture(servicegenai.GenerationStatusSucceeded, nil),
			},
			expectErr: core.ErrGenerationOutputInvalid,
		},
		{
			name:      "Error/MalformedResponsesOutput",
			request:   validRequest,
			callGenAI: true,
			genaiResponse: &servicegenai.GenerationGetResponse{
				Generation: generationFixture(
					servicegenai.GenerationStatusSucceeded,
					json.RawMessage(`{`),
				),
			},
			expectErr: core.ErrGenerationOutputInvalid,
		},
		{
			name:      "Error/EnvelopeFieldPrivacy",
			request:   validRequest,
			callGenAI: true,
			genaiResponse: &servicegenai.GenerationGetResponse{
				Generation: generationFixture(
					servicegenai.GenerationStatusSucceeded,
					responsesOutputText(t, `{"`+privateGenerationOutput+`":true}`),
				),
			},
			expectErr:         core.ErrGenerationOutputInvalid,
			expectErrExcludes: privateGenerationOutput,
		},
		{
			name:      "Error/NoOutputText",
			request:   validRequest,
			callGenAI: true,
			genaiResponse: &servicegenai.GenerationGetResponse{
				Generation: generationFixture(
					servicegenai.GenerationStatusSucceeded,
					json.RawMessage(`{}`),
				),
			},
			expectErr: core.ErrGenerationOutputInvalid,
		},
		{
			name:      "Error/MultipleEnvelopeValues",
			request:   validRequest,
			callGenAI: true,
			genaiResponse: &servicegenai.GenerationGetResponse{
				Generation: generationFixture(
					servicegenai.GenerationStatusSucceeded,
					responsesOutputText(t, generationEnvelope(t, manuscriptValue)+" {}"),
				),
			},
			expectErr: core.ErrGenerationOutputInvalid,
		},
		{
			name:      "Error/UnknownStatus",
			request:   validRequest,
			callGenAI: true,
			genaiResponse: &servicegenai.GenerationGetResponse{
				Generation: generationFixture(servicegenai.GenerationStatus(99), nil),
			},
			expectErr: core.ErrGenerationStatusUnknown,
		},
		{
			name:          "Error/OwnerMismatch",
			request:       validRequest,
			callGenAI:     true,
			genaiResponse: &servicegenai.GenerationGetResponse{Generation: mismatchedOwner},
			expectErr:     core.ErrGenerationResponseInvalid,
		},
		{
			name:          "Error/PurposeMismatch",
			request:       validRequest,
			callGenAI:     true,
			genaiResponse: &servicegenai.GenerationGetResponse{Generation: mismatchedPurpose},
			expectErr:     core.ErrGenerationResponseInvalid,
		},
		{
			name:          "Error/IDMismatch",
			request:       validRequest,
			callGenAI:     true,
			genaiResponse: &servicegenai.GenerationGetResponse{Generation: mismatchedID},
			expectErr:     core.ErrGenerationResponseInvalid,
		},
		{
			name:          "Error/InvalidGenerationID",
			request:       validRequest,
			callGenAI:     true,
			genaiResponse: &servicegenai.GenerationGetResponse{Generation: invalidGenerationID},
			expectErr:     core.ErrGenerationResponseInvalid,
		},
		{
			name:          "Error/InvalidOwnerID",
			request:       validRequest,
			callGenAI:     true,
			genaiResponse: &servicegenai.GenerationGetResponse{Generation: invalidOwnerID},
			expectErr:     core.ErrGenerationResponseInvalid,
		},
		{
			name:          "Error/MissingCreatedAt",
			request:       validRequest,
			callGenAI:     true,
			genaiResponse: &servicegenai.GenerationGetResponse{Generation: missingCreatedAt},
			expectErr:     core.ErrGenerationResponseInvalid,
		},
		{
			name:          "Error/InvalidSettledAt",
			request:       validRequest,
			callGenAI:     true,
			genaiResponse: &servicegenai.GenerationGetResponse{Generation: invalidSettledAt},
			expectErr:     core.ErrGenerationResponseInvalid,
		},
		{
			name:          "Error/InvalidTimestamp",
			request:       validRequest,
			callGenAI:     true,
			genaiResponse: &servicegenai.GenerationGetResponse{Generation: invalidTime},
			expectErr:     core.ErrGenerationResponseInvalid,
		},
		{
			name:      "Error/Refusal",
			request:   validRequest,
			callGenAI: true,
			genaiResponse: &servicegenai.GenerationGetResponse{
				Generation: generationFixture(
					servicegenai.GenerationStatusSucceeded,
					json.RawMessage(
						`{"output":[{"content":[{"type":"refusal","refusal":"`+
							privateGenerationOutput+`"}]}]}`,
					),
				),
			},
			expectErr:         core.ErrGenerationRefused,
			expectErrExcludes: privateGenerationOutput,
		},
		{
			// A refusal outranks any text beside it, so the caller learns the
			// generation was refused rather than that its output was malformed.
			name:      "Error/RefusalBesideOutputText",
			request:   validRequest,
			callGenAI: true,
			genaiResponse: &servicegenai.GenerationGetResponse{
				Generation: generationFixture(
					servicegenai.GenerationStatusSucceeded,
					json.RawMessage(
						`{"output_text":"{}","output":[{"content":[`+
							`{"type":"output_text","text":"{}"},`+
							`{"type":"refusal","refusal":"`+privateGenerationOutput+`"}]}]}`,
					),
				),
			},
			expectErr:         core.ErrGenerationRefused,
			expectErrExcludes: privateGenerationOutput,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			engineVersionDao := coremocks.NewMockEngineVersionSelectDao(t)
			genai := servicegenaimocks.NewMockClient(t)

			if testCase.callGenAI {
				genai.EXPECT().
					GenerationGet(mock.Anything, &servicegenai.GenerationGetRequest{
						Id:      testCase.request.ID.String(),
						OwnerId: testCase.request.Actor.UserID.String(),
					}).
					Return(testCase.genaiResponse, testCase.genaiErr)
			}

			if testCase.engineResponse != nil || testCase.engineErr != nil {
				engineVersionDao.EXPECT().
					Exec(mock.Anything, &dao.EngineVersionSelectRequest{ID: engineVersionID}).
					Return(testCase.engineResponse, testCase.engineErr)
			}

			result, err := core.NewGenerationGet(engineVersionDao, genai).
				Exec(t.Context(), testCase.request)

			require.ErrorIs(t, err, testCase.expectErr)

			if testCase.expectErrExcludes != "" {
				require.NotContains(t, err.Error(), testCase.expectErrExcludes)
			}

			require.Equal(t, testCase.expect, result)
		})
	}
}
