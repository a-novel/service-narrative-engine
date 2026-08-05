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
)

func TestGenerationGet(t *testing.T) {
	t.Parallel()

	const privateProviderError = "private provider details"

	errAccess := errors.New("access failure")
	errGenAI := errors.New("genai failure")
	proposal := json.RawMessage(`{"shape":"belongs to the client"}`)
	validRequest := &core.GenerationGetRequest{
		Actor: core.Actor{UserID: ownerID}, ProjectID: projectID, ID: generationID,
	}
	failed := generationFixture(servicegenai.GenerationStatusFailed, nil)
	failed.Error = privateProviderError
	expectFailed := expectedGeneration(core.GenerationStatusFailed, nil)
	expectFailed.Failure = "generation failed"
	wrongOwner := generationFixture(servicegenai.GenerationStatusPending, nil)
	wrongOwner.OwnerId = uuid.MustParse("00000000-0000-0000-0000-000000000099").String()
	wrongID := generationFixture(servicegenai.GenerationStatusPending, nil)
	wrongID.Id = uuid.MustParse("00000000-0000-0000-0000-000000000699").String()
	unknownStatus := generationFixture(servicegenai.GenerationStatus(99), nil)

	testCases := []struct {
		name string

		request    *core.GenerationGetRequest
		accessErr  error
		response   *servicegenai.GenerationGetResponse
		genaiErr   error
		callAccess bool
		callGenAI  bool

		expect    *core.Generation
		expectErr error
	}{
		{
			name: "Success/Pending", request: validRequest, callAccess: true, callGenAI: true,
			response: &servicegenai.GenerationGetResponse{
				Generation: generationFixture(servicegenai.GenerationStatusPending, nil),
			},
			expect: expectedGeneration(core.GenerationStatusPending, nil),
		},
		{
			name: "Success/OpaqueJSONProposal", request: validRequest, callAccess: true, callGenAI: true,
			response: &servicegenai.GenerationGetResponse{
				Generation: generationFixture(
					servicegenai.GenerationStatusSucceeded,
					responsesOutput(t, proposal),
				),
			},
			expect: expectedGeneration(core.GenerationStatusSucceeded, proposal),
		},
		{
			name: "Success/ProviderErrorIsOpaque", request: validRequest,
			callAccess: true, callGenAI: true,
			response: &servicegenai.GenerationGetResponse{Generation: failed},
			expect:   expectFailed,
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
			name: "Error/NotFound", request: validRequest, callAccess: true, callGenAI: true,
			genaiErr:  status.Error(codes.NotFound, "not found"),
			expectErr: core.ErrGenerationNotFound,
		},
		{
			name: "Error/GenAI", request: validRequest, callAccess: true, callGenAI: true,
			genaiErr: errGenAI, expectErr: errGenAI,
		},
		{
			name: "Error/MissingResponse", request: validRequest,
			callAccess: true, callGenAI: true, expectErr: core.ErrGenerationResponseInvalid,
		},
		{
			name: "Error/WrongOwner", request: validRequest, callAccess: true, callGenAI: true,
			response:  &servicegenai.GenerationGetResponse{Generation: wrongOwner},
			expectErr: core.ErrGenerationResponseInvalid,
		},
		{
			name: "Error/WrongID", request: validRequest, callAccess: true, callGenAI: true,
			response:  &servicegenai.GenerationGetResponse{Generation: wrongID},
			expectErr: core.ErrGenerationResponseInvalid,
		},
		{
			name: "Error/UnknownStatus", request: validRequest, callAccess: true, callGenAI: true,
			response:  &servicegenai.GenerationGetResponse{Generation: unknownStatus},
			expectErr: core.ErrGenerationStatusUnknown,
		},
		{
			name: "Error/InvalidProposalJSON", request: validRequest, callAccess: true, callGenAI: true,
			response: &servicegenai.GenerationGetResponse{Generation: generationFixture(
				servicegenai.GenerationStatusSucceeded,
				responsesOutputText(t, "{"),
			)},
			expectErr: core.ErrGenerationOutputInvalid,
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
					Return(projectFixture(), testCase.accessErr)
			}

			if testCase.callGenAI {
				genai.EXPECT().
					GenerationGet(mock.Anything, &servicegenai.GenerationGetRequest{
						Id: testCase.request.ID.String(), OwnerId: testCase.request.Actor.UserID.String(),
					}).
					Return(testCase.response, testCase.genaiErr)
			}

			result, err := core.NewGenerationGet(projectAccess, genai).
				Exec(t.Context(), testCase.request)

			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			if result != nil {
				require.NotContains(t, result.Failure, privateProviderError)
			}
		})
	}
}
