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

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/lib"
)

func TestGenerationSubmit(t *testing.T) {
	t.Parallel()

	errAccess := errors.New("access failure")
	errGateway := errors.New("generation gateway failure")
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
	validResult := &lib.GenerationSubmitGatewayResult{
		Generation: generationFixture(lib.GenerationStatusSucceeded, string(proposal)),
		Created:    true,
	}
	invalidOwner := generationFixture(lib.GenerationStatusPending, "")
	invalidOwner.OwnerID = uuid.MustParse("00000000-0000-0000-0000-000000000099")

	testCases := []struct {
		name string

		request        *core.GenerationSubmitRequest
		accessErr      error
		gatewayResult  *lib.GenerationSubmitGatewayResult
		gatewayErr     error
		callAccess     bool
		callGateway    bool
		inspectPayload bool

		expect    *core.GenerationSubmitResult
		expectErr error
	}{
		{
			name:           "Success/ClientComposedAndSchemaOpaque",
			request:        validRequest,
			callAccess:     true,
			callGateway:    true,
			inspectPayload: true,
			gatewayResult:  validResult,
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
			callAccess: true, callGateway: true,
			gatewayResult: &lib.GenerationSubmitGatewayResult{
				Generation: generationFixture(lib.GenerationStatusPending, ""),
				Created:    true,
			},
			expect: &core.GenerationSubmitResult{
				Generation: expectedGeneration(core.GenerationStatusPending, nil),
				Created:    true,
			},
		},
		{
			name: "Error/InvalidRequest", request: &core.GenerationSubmitRequest{},
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
				Input: contentDocumentOfSize(600_000), Context: contentDocumentOfSize(600_000),
				OutputSchema: json.RawMessage(`{}`),
			},
			callAccess: true, expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/Conflict", request: validRequest,
			callAccess: true, callGateway: true,
			gatewayErr: lib.ErrGenerationConflict, expectErr: core.ErrGenerationConflict,
		},
		{
			name: "Error/Gateway", request: validRequest,
			callAccess: true, callGateway: true,
			gatewayErr: errGateway, expectErr: errGateway,
		},
		{
			name: "Error/MissingResponse", request: validRequest,
			callAccess: true, callGateway: true,
			expectErr: core.ErrGenerationResponseInvalid,
		},
		{
			name: "Error/MissingGeneration", request: validRequest,
			callAccess: true, callGateway: true,
			gatewayResult: &lib.GenerationSubmitGatewayResult{},
			expectErr:     core.ErrGenerationResponseInvalid,
		},
		{
			name: "Error/InvalidResponseOwner", request: validRequest,
			callAccess: true, callGateway: true,
			gatewayResult: &lib.GenerationSubmitGatewayResult{Generation: invalidOwner},
			expectErr:     core.ErrGenerationResponseInvalid,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			projectAccess := coremocks.NewMockProjectAccessService(t)
			gateway := coremocks.NewMockGenerationSubmitGateway(t)

			if testCase.callAccess {
				projectAccess.EXPECT().
					Exec(mock.Anything, &core.ProjectAccessRequest{
						Actor: testCase.request.Actor, ProjectID: testCase.request.ProjectID,
					}).
					Return(projectFixture(), testCase.accessErr).
					Once()
			}

			if testCase.callGateway {
				gateway.EXPECT().
					Exec(mock.Anything, mock.Anything).
					RunAndReturn(func(
						_ context.Context,
						request *lib.GenerationSubmitGatewayRequest,
					) (*lib.GenerationSubmitGatewayResult, error) {
						require.Equal(t, ownerID, request.OwnerID)
						require.Equal(t, core.GenerationPurposeStudio, request.Purpose)
						require.Len(t, request.IdempotencyKey, 64)
						require.NotEqual(t, testCase.request.IdempotencyKey, request.IdempotencyKey)
						require.EqualValues(t, 2, request.MaxAttempts)

						if testCase.inspectPayload {
							assertGenerationProviderPayload(t, request.Request, testCase.request)
						}

						return testCase.gatewayResult, testCase.gatewayErr
					}).
					Once()
			}

			result, err := core.NewGenerationSubmit(projectAccess, gateway).
				Exec(t.Context(), testCase.request)

			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)
			projectAccess.AssertExpectations(t)
			gateway.AssertExpectations(t)
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
		Model        string `json:"model"`
		Instructions string `json:"instructions"`
		Input        string `json:"input"`
		Reasoning    struct {
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
