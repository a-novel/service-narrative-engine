package core_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/lib"
)

func TestGenerationGet(t *testing.T) {
	t.Parallel()

	errAccess := errors.New("access failure")
	errGateway := errors.New("generation gateway failure")
	proposal := json.RawMessage(`{"shape":"belongs to the client"}`)
	validRequest := &core.GenerationGetRequest{
		Actor: core.Actor{UserID: ownerID}, ProjectID: projectID, ID: generationID,
	}
	failed := generationFixture(lib.GenerationStatusFailed, "")
	failed.Failed = true
	expectFailed := expectedGeneration(core.GenerationStatusFailed, nil)
	expectFailed.Failure = "generation failed"
	wrongOwner := generationFixture(lib.GenerationStatusPending, "")
	wrongOwner.OwnerID = uuid.MustParse("00000000-0000-0000-0000-000000000099")
	wrongID := generationFixture(lib.GenerationStatusPending, "")
	wrongID.ID = uuid.MustParse("00000000-0000-0000-0000-000000000699")
	wrongPurpose := generationFixture(lib.GenerationStatusPending, "")
	wrongPurpose.Purpose = "other"
	unknownStatus := generationFixture(lib.GenerationStatus("unknown"), "")

	testCases := []struct {
		name string

		request     *core.GenerationGetRequest
		accessErr   error
		response    *lib.Generation
		gatewayErr  error
		callAccess  bool
		callGateway bool

		expect    *core.Generation
		expectErr error
	}{
		{
			name: "Success/Pending", request: validRequest, callAccess: true, callGateway: true,
			response: generationFixture(lib.GenerationStatusPending, ""),
			expect:   expectedGeneration(core.GenerationStatusPending, nil),
		},
		{
			name: "Success/OpaqueJSONProposal", request: validRequest,
			callAccess: true, callGateway: true,
			response: generationFixture(lib.GenerationStatusSucceeded, string(proposal)),
			expect:   expectedGeneration(core.GenerationStatusSucceeded, proposal),
		},
		{
			name: "Success/ProviderErrorIsOpaque", request: validRequest,
			callAccess: true, callGateway: true, response: failed, expect: expectFailed,
		},
		{
			name: "Error/InvalidRequest", request: &core.GenerationGetRequest{},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/ProjectAccess", request: validRequest,
			callAccess: true, accessErr: errAccess, expectErr: errAccess,
		},
		{
			name: "Error/NotFound", request: validRequest,
			callAccess: true, callGateway: true,
			gatewayErr: lib.ErrGenerationNotFound, expectErr: core.ErrGenerationNotFound,
		},
		{
			name: "Error/Gateway", request: validRequest,
			callAccess: true, callGateway: true, gatewayErr: errGateway, expectErr: errGateway,
		},
		{
			name: "Error/MissingResponse", request: validRequest,
			callAccess: true, callGateway: true, expectErr: core.ErrGenerationResponseInvalid,
		},
		{
			name: "Error/WrongOwner", request: validRequest,
			callAccess: true, callGateway: true, response: wrongOwner,
			expectErr: core.ErrGenerationResponseInvalid,
		},
		{
			name: "Error/WrongID", request: validRequest,
			callAccess: true, callGateway: true, response: wrongID,
			expectErr: core.ErrGenerationResponseInvalid,
		},
		{
			name: "Error/WrongPurpose", request: validRequest,
			callAccess: true, callGateway: true, response: wrongPurpose,
			expectErr: core.ErrGenerationResponseInvalid,
		},
		{
			name: "Error/UnknownStatus", request: validRequest,
			callAccess: true, callGateway: true, response: unknownStatus,
			expectErr: core.ErrGenerationStatusUnknown,
		},
		{
			name: "Error/InvalidProposalJSON", request: validRequest,
			callAccess: true, callGateway: true,
			response:  generationFixture(lib.GenerationStatusSucceeded, "{"),
			expectErr: core.ErrGenerationOutputInvalid,
		},
		{
			name: "Error/ProposalOverLimit", request: validRequest,
			callAccess: true, callGateway: true,
			response: generationFixture(
				lib.GenerationStatusSucceeded,
				string(contentDocumentOfSize((1<<20)+1)),
			),
			expectErr: core.ErrGenerationOutputInvalid,
		},
		{
			name: "Error/Refused", request: validRequest,
			callAccess: true, callGateway: true,
			gatewayErr: lib.ErrGenerationRefused, expectErr: core.ErrGenerationRefused,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			projectAccess := coremocks.NewMockProjectAccessService(t)
			gateway := coremocks.NewMockGenerationGetGateway(t)

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
					Exec(mock.Anything, &lib.GenerationGetGatewayRequest{
						ID: testCase.request.ID, OwnerID: testCase.request.Actor.UserID,
					}).
					Return(testCase.response, testCase.gatewayErr).
					Once()
			}

			result, err := core.NewGenerationGet(projectAccess, gateway).
				Exec(t.Context(), testCase.request)

			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)
			projectAccess.AssertExpectations(t)
			gateway.AssertExpectations(t)
		})
	}
}
