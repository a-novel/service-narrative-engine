package core_test

import (
	"encoding/json"
	"errors"
	"io"
	"testing"

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

type generationWatchStream struct {
	grpc.ClientStream

	responses []*servicegenai.GenerationWatchResponse
	err       error
	index     int
}

func (stream *generationWatchStream) Recv() (*servicegenai.GenerationWatchResponse, error) {
	if stream.index < len(stream.responses) {
		response := stream.responses[stream.index]
		stream.index++

		return response, nil
	}

	if stream.err != nil {
		err := stream.err
		stream.err = nil

		return nil, err
	}

	return nil, io.EOF
}

func TestGenerationWatch(t *testing.T) {
	t.Parallel()

	errAccess := errors.New("access failure")
	errStream := errors.New("stream failure")
	proposal := map[string]any{"freeform": []any{"client", "shape"}}
	validRequest := &core.GenerationWatchRequest{
		Actor: core.Actor{UserID: ownerID}, ProjectID: projectID, ID: generationID,
	}

	testCases := []struct {
		name string

		request    *core.GenerationWatchRequest
		accessErr  error
		responses  []*servicegenai.GenerationWatchResponse
		streamErr  error
		initialErr error
		nilStream  bool
		callAccess bool
		callGenAI  bool

		expect    *core.Generation
		expectErr error
	}{
		{
			name: "Success/Succeeded", request: validRequest, callAccess: true, callGenAI: true,
			responses: []*servicegenai.GenerationWatchResponse{
				{Generation: generationFixture(servicegenai.GenerationStatusPending, nil)},
				{Generation: generationFixture(servicegenai.GenerationStatusRunning, nil)},
				{Generation: generationFixture(
					servicegenai.GenerationStatusSucceeded,
					responsesOutput(t, proposal),
				)},
			},
			expect: expectedGeneration(
				core.GenerationStatusSucceeded,
				json.RawMessage(`{"freeform":["client","shape"]}`),
			),
		},
		{
			name: "Success/Failed", request: validRequest, callAccess: true, callGenAI: true,
			responses: []*servicegenai.GenerationWatchResponse{{
				Generation: generationFixture(servicegenai.GenerationStatusFailed, nil),
			}},
			expect: expectedGeneration(core.GenerationStatusFailed, nil),
		},
		{
			name: "Error/InvalidRequest", request: &core.GenerationWatchRequest{},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/ProjectAccess", request: validRequest,
			callAccess: true, accessErr: errAccess, expectErr: errAccess,
		},
		{
			name: "Error/InitialNotFound", request: validRequest, callAccess: true, callGenAI: true,
			initialErr: status.Error(codes.NotFound, "not found"),
			expectErr:  core.ErrGenerationNotFound,
		},
		{
			name: "Error/Initial", request: validRequest, callAccess: true, callGenAI: true,
			initialErr: errStream, expectErr: errStream,
		},
		{
			name: "Error/MissingStream", request: validRequest, callAccess: true, callGenAI: true,
			nilStream: true, expectErr: core.ErrGenerationResponseInvalid,
		},
		{
			name: "Error/Closed", request: validRequest, callAccess: true, callGenAI: true,
			responses: []*servicegenai.GenerationWatchResponse{{
				Generation: generationFixture(servicegenai.GenerationStatusPending, nil),
			}},
			expectErr: core.ErrGenerationWatchClosed,
		},
		{
			name: "Error/ReceiveNotFound", request: validRequest, callAccess: true, callGenAI: true,
			streamErr: status.Error(codes.NotFound, "not found"),
			expectErr: core.ErrGenerationNotFound,
		},
		{
			name: "Error/Receive", request: validRequest, callAccess: true, callGenAI: true,
			streamErr: errStream, expectErr: errStream,
		},
		{
			name: "Error/MissingResponse", request: validRequest, callAccess: true, callGenAI: true,
			responses: []*servicegenai.GenerationWatchResponse{nil},
			expectErr: core.ErrGenerationResponseInvalid,
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
				var stream grpc.ServerStreamingClient[servicegenai.GenerationWatchResponse]
				if testCase.initialErr == nil && !testCase.nilStream {
					stream = &generationWatchStream{
						responses: testCase.responses,
						err:       testCase.streamErr,
					}
				}

				genai.EXPECT().
					GenerationWatch(mock.Anything, &servicegenai.GenerationWatchRequest{
						Id: testCase.request.ID.String(), OwnerId: testCase.request.Actor.UserID.String(),
					}).
					Return(stream, testCase.initialErr)
			}

			result, err := core.NewGenerationWatch(projectAccess, genai).
				Exec(t.Context(), testCase.request)

			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)
		})
	}
}
