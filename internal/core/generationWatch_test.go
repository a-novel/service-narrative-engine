package core_test

import (
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/lib"
)

type generationWatchGatewayStream struct {
	responses []*lib.Generation
	err       error
	index     int
	closed    bool
}

func (stream *generationWatchGatewayStream) Recv() (*lib.Generation, error) {
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

func (stream *generationWatchGatewayStream) Close() {
	stream.closed = true
}

func TestGenerationWatch(t *testing.T) {
	t.Parallel()

	errAccess := errors.New("access failure")
	errStream := errors.New("stream failure")
	proposal := json.RawMessage(`{"freeform":["client","shape"]}`)
	validRequest := &core.GenerationWatchRequest{
		Actor: core.Actor{UserID: ownerID}, ProjectID: projectID, ID: generationID,
	}

	testCases := []struct {
		name string

		request     *core.GenerationWatchRequest
		accessErr   error
		responses   []*lib.Generation
		streamErr   error
		initialErr  error
		nilStream   bool
		callAccess  bool
		callGateway bool

		expect    *core.Generation
		expectErr error
	}{
		{
			name: "Success/Succeeded", request: validRequest,
			callAccess: true, callGateway: true,
			responses: []*lib.Generation{
				generationFixture(lib.GenerationStatusPending, ""),
				generationFixture(lib.GenerationStatusRunning, ""),
				generationFixture(lib.GenerationStatusSucceeded, string(proposal)),
			},
			expect: expectedGeneration(core.GenerationStatusSucceeded, proposal),
		},
		{
			name: "Success/Failed", request: validRequest,
			callAccess: true, callGateway: true,
			responses: []*lib.Generation{
				generationFixture(lib.GenerationStatusFailed, ""),
			},
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
			name: "Error/InitialNotFound", request: validRequest,
			callAccess: true, callGateway: true,
			initialErr: lib.ErrGenerationNotFound, expectErr: core.ErrGenerationNotFound,
		},
		{
			name: "Error/Initial", request: validRequest,
			callAccess: true, callGateway: true,
			initialErr: errStream, expectErr: errStream,
		},
		{
			name: "Error/MissingStream", request: validRequest,
			callAccess: true, callGateway: true, nilStream: true,
			expectErr: core.ErrGenerationResponseInvalid,
		},
		{
			name: "Error/Closed", request: validRequest,
			callAccess: true, callGateway: true,
			responses: []*lib.Generation{
				generationFixture(lib.GenerationStatusPending, ""),
			},
			expectErr: core.ErrGenerationWatchClosed,
		},
		{
			name: "Error/ReceiveNotFound", request: validRequest,
			callAccess: true, callGateway: true,
			streamErr: lib.ErrGenerationNotFound, expectErr: core.ErrGenerationNotFound,
		},
		{
			name: "Error/Receive", request: validRequest,
			callAccess: true, callGateway: true,
			streamErr: errStream, expectErr: errStream,
		},
		{
			name: "Error/MissingResponse", request: validRequest,
			callAccess: true, callGateway: true,
			responses: []*lib.Generation{nil},
			expectErr: core.ErrGenerationResponseInvalid,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			projectAccess := coremocks.NewMockProjectAccessService(t)
			gateway := coremocks.NewMockGenerationWatchGateway(t)

			if testCase.callAccess {
				projectAccess.EXPECT().
					Exec(mock.Anything, &core.ProjectAccessRequest{
						Actor: testCase.request.Actor, ProjectID: testCase.request.ProjectID,
					}).
					Return(projectFixture(), testCase.accessErr).
					Once()
			}

			var (
				stream        *generationWatchGatewayStream
				gatewayStream lib.GenerationWatchGatewayStream
			)

			if testCase.callGateway {
				if testCase.initialErr == nil && !testCase.nilStream {
					stream = &generationWatchGatewayStream{
						responses: testCase.responses,
						err:       testCase.streamErr,
					}
					gatewayStream = stream
				}

				gateway.EXPECT().
					Exec(mock.Anything, &lib.GenerationWatchGatewayRequest{
						ID: testCase.request.ID, OwnerID: testCase.request.Actor.UserID,
					}).
					Return(gatewayStream, testCase.initialErr).
					Once()
			}

			result, err := core.NewGenerationWatch(projectAccess, gateway).
				Exec(t.Context(), testCase.request)

			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			if stream != nil {
				require.True(t, stream.closed)
			}

			projectAccess.AssertExpectations(t)
			gateway.AssertExpectations(t)
		})
	}
}
