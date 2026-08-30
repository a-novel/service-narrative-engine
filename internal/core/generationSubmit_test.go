package core_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicegenai "github.com/a-novel/service-genai/pkg/go"
	servicegenaimocks "github.com/a-novel/service-genai/pkg/go/mocks"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
)

func TestGenerationSubmit(t *testing.T) {
	t.Parallel()

	errAccess := errors.New("access failure")
	errGenAI := errors.New("genai failure")
	proposal := json.RawMessage(`{"clientOwns":"this shape","extra":[1,2,3]}`)
	validRequest := &core.GenerationSubmitRequest{
		Actor:          core.Actor{UserID: ownerID},
		ProjectID:      projectID,
		IdempotencyKey: "retry-1",
		Instructions:   "Complete only the requested story fragment.",
		Input:          json.RawMessage(`{"partial":"scene"}`),
		Context:        json.RawMessage(`{"digest":"prior context","steps":[1,2]}`),
		OutputSchema:   json.RawMessage(`{"type":"string"}`),
	}
	validResponse := &servicegenai.GenerationSubmitResponse{
		Generation: generationFixture(
			servicegenai.GenerationStatusSucceeded,
			responsesOutput(t, proposal),
		),
		Created: true,
	}
	invalidOwnerResponse := generationFixture(servicegenai.GenerationStatusPending, nil)
	invalidOwnerResponse.OwnerId = uuid.MustParse(
		"00000000-0000-0000-0000-000000000099",
	).String()

	testCases := []struct {
		name string

		request        *core.GenerationSubmitRequest
		accessErr      error
		genaiResponse  *servicegenai.GenerationSubmitResponse
		genaiErr       error
		callAccess     bool
		callGenAI      bool
		inspectPayload bool

		expect    *core.GenerationSubmitResult
		expectErr error
	}{
		{
			name:           "Success/ClientComposedAndSchemaOpaque",
			request:        validRequest,
			callAccess:     true,
			callGenAI:      true,
			inspectPayload: true,
			genaiResponse:  validResponse,
			expect: &core.GenerationSubmitResult{
				Generation: expectedGeneration(core.GenerationStatusSucceeded, proposal),
				Created:    true,
			},
		},
		{
			name: "Success/ValidJSONButInvalidJSONSchemaIsForwarded",
			request: &core.GenerationSubmitRequest{
				Actor: core.Actor{UserID: ownerID}, ProjectID: projectID,
				IdempotencyKey: "retry-invalid-schema",
				Instructions:   "Generate.", Input: json.RawMessage(`null`),
				Context: json.RawMessage(`[]`), OutputSchema: json.RawMessage(`"not-a-schema"`),
			},
			callAccess: true, callGenAI: true, genaiResponse: &servicegenai.GenerationSubmitResponse{
				Generation: generationFixture(servicegenai.GenerationStatusPending, nil),
				Created:    true,
			},
			expect: &core.GenerationSubmitResult{
				Generation: expectedGeneration(core.GenerationStatusPending, nil),
				Created:    true,
			},
		},
		{
			name:      "Error/InvalidRequest",
			request:   &core.GenerationSubmitRequest{},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/InstructionsOverLimit",
			request: &core.GenerationSubmitRequest{
				Actor: core.Actor{UserID: ownerID}, ProjectID: projectID,
				IdempotencyKey: "retry", Instructions: strings.Repeat("i", 32_769),
				Input: json.RawMessage(`{}`), Context: json.RawMessage(`{}`),
				OutputSchema: json.RawMessage(`{}`),
			},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/ProjectAccessBeforeJSONValidation",
			request: &core.GenerationSubmitRequest{
				Actor: core.Actor{UserID: ownerID}, ProjectID: projectID,
				IdempotencyKey: "retry", Instructions: "Generate.",
				Input: json.RawMessage(`{`), Context: json.RawMessage(`{}`),
				OutputSchema: json.RawMessage(`{}`),
			},
			callAccess: true, accessErr: errAccess, expectErr: errAccess,
		},
		{
			name: "Error/MalformedInput",
			request: &core.GenerationSubmitRequest{
				Actor: core.Actor{UserID: ownerID}, ProjectID: projectID,
				IdempotencyKey: "retry", Instructions: "Generate.",
				Input: json.RawMessage(`{`), Context: json.RawMessage(`{}`),
				OutputSchema: json.RawMessage(`{}`),
			},
			callAccess: true, expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/ComponentOverLimit",
			request: &core.GenerationSubmitRequest{
				Actor: core.Actor{UserID: ownerID}, ProjectID: projectID,
				IdempotencyKey: "retry", Instructions: "Generate.",
				Input:   contentDocumentOfSize((1 << 20) + 1),
				Context: json.RawMessage(`{}`), OutputSchema: json.RawMessage(`{}`),
			},
			callAccess: true, expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/AggregatePayloadOverLimit",
			request: &core.GenerationSubmitRequest{
				Actor: core.Actor{UserID: ownerID}, ProjectID: projectID,
				IdempotencyKey: "retry", Instructions: "Generate.",
				Input:        contentDocumentOfSize(600_000),
				Context:      contentDocumentOfSize(600_000),
				OutputSchema: json.RawMessage(`{}`),
			},
			callAccess: true, expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/Conflict", request: validRequest,
			callAccess: true, callGenAI: true,
			genaiErr:  status.Error(codes.AlreadyExists, "conflict"),
			expectErr: core.ErrGenerationConflict,
		},
		{
			name: "Error/GenAI", request: validRequest,
			callAccess: true, callGenAI: true, genaiErr: errGenAI, expectErr: errGenAI,
		},
		{
			name: "Error/MissingResponse", request: validRequest,
			callAccess: true, callGenAI: true, expectErr: core.ErrGenerationResponseInvalid,
		},
		{
			name: "Error/InvalidResponseOwner", request: validRequest,
			callAccess: true, callGenAI: true,
			genaiResponse: &servicegenai.GenerationSubmitResponse{Generation: invalidOwnerResponse},
			expectErr:     core.ErrGenerationResponseInvalid,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			projectAccess := coremocks.NewMockProjectAccessService(t)
			genai := servicegenaimocks.NewMockClient(t)

			if testCase.callAccess {
				projectAccess.EXPECT().
					Exec(mock.Anything, &core.ProjectAccessRequest{
						Actor: testCase.request.Actor, ProjectID: testCase.request.ProjectID,
					}).
					Return(projectFixture(), testCase.accessErr).
					Once()
			}

			if testCase.callGenAI {
				genai.EXPECT().
					GenerationSubmit(mock.Anything, mock.Anything).
					RunAndReturn(func(
						_ context.Context,
						request *servicegenai.GenerationSubmitRequest,
						_ ...grpc.CallOption,
					) (*servicegenai.GenerationSubmitResponse, error) {
						require.Equal(t, ownerID.String(), request.GetOwnerId())
						require.Equal(t, core.GenerationPurposeStudio, request.GetPurpose())
						require.Len(t, request.GetIdempotencyKey(), 64)
						require.NotEqual(t, testCase.request.IdempotencyKey, request.GetIdempotencyKey())
						require.EqualValues(t, 2, request.GetMaxAttempts())

						if testCase.inspectPayload {
							assertGenerationProviderPayload(t, request.GetRequest(), testCase.request)
						}

						return testCase.genaiResponse, testCase.genaiErr
					})
			}

			result, err := core.NewGenerationSubmit(projectAccess, genai).
				Exec(t.Context(), testCase.request)

			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)
		})
	}
}

func assertGenerationProviderPayload(
	t *testing.T,
	payload []byte,
	request *core.GenerationSubmitRequest,
) {
	t.Helper()

	var providerRequest struct {
		Model            string `json:"model"`
		MaxOutputTokens  int64  `json:"max_output_tokens"`
		SafetyIdentifier string `json:"safety_identifier"`
		Instructions     string `json:"instructions"`
		Input            string `json:"input"`
		Reasoning        struct {
			Effort string `json:"effort"`
		} `json:"reasoning"`
		Text struct {
			Format struct {
				Type   string          `json:"type"`
				Name   string          `json:"name"`
				Schema json.RawMessage `json:"schema"`
				Strict bool            `json:"strict"`
			} `json:"format"`
		} `json:"text"`
	}

	require.NoError(t, json.Unmarshal(payload, &providerRequest))
	require.Equal(t, core.GenerationModelDefault, providerRequest.Model)
	require.Equal(t, core.GenerationReasoningEffortDefault, providerRequest.Reasoning.Effort)
	require.Equal(t, core.GenerationMaxOutputTokensDefault, providerRequest.MaxOutputTokens)
	require.Equal(
		t,
		"c5c62cc8237a1364feed2ff5e8daeaa0c03a554d9d75f60605fda0933b333784",
		providerRequest.SafetyIdentifier,
	)
	require.NotContains(t, providerRequest.SafetyIdentifier, ownerID.String())
	require.Equal(t, request.Instructions, providerRequest.Instructions)
	require.Equal(t, "json_schema", providerRequest.Text.Format.Type)
	require.Equal(t, "project_content_output", providerRequest.Text.Format.Name)
	require.True(t, providerRequest.Text.Format.Strict)
	require.JSONEq(t, string(request.OutputSchema), string(providerRequest.Text.Format.Schema))

	var inputDocument struct {
		Input   json.RawMessage `json:"input"`
		Context json.RawMessage `json:"context"`
	}
	require.NoError(t, json.Unmarshal([]byte(providerRequest.Input), &inputDocument))
	require.JSONEq(t, string(request.Input), string(inputDocument.Input))
	require.JSONEq(t, string(request.Context), string(inputDocument.Context))
}
