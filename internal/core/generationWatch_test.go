package core_test

import (
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
	"github.com/a-novel/service-narrative-engine/internal/dao"
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

	errFoo := errors.New("foo")
	validRequest := &core.GenerationWatchRequest{
		Actor: core.Actor{UserID: ownerID},
		ID:    generationID,
	}

	testCases := []struct {
		name string

		request      *core.GenerationWatchRequest
		responses    []*servicegenai.GenerationWatchResponse
		streamErr    error
		initialErr   error
		callGenAI    bool
		engineResult *dao.EngineVersion

		expect    *core.Generation
		expectErr error
	}{
		{
			name:      "Success/Succeeded",
			request:   validRequest,
			callGenAI: true,
			responses: []*servicegenai.GenerationWatchResponse{
				{Generation: generationFixture(servicegenai.GenerationStatusPending, nil)},
				{Generation: generationFixture(servicegenai.GenerationStatusRunning, nil)},
				{Generation: generationFixture(
					servicegenai.GenerationStatusSucceeded,
					responsesOutput(t, manuscriptValue),
				)},
			},
			engineResult: engineVersionFixture(),
			expect:       expectedGeneration(core.GenerationStatusSucceeded, &manuscriptValue),
		},
		{
			name:      "Success/Failed",
			request:   validRequest,
			callGenAI: true,
			responses: []*servicegenai.GenerationWatchResponse{{
				Generation: generationFixture(servicegenai.GenerationStatusFailed, nil),
			}},
			expect: expectedGeneration(core.GenerationStatusFailed, nil),
		},
		{
			name:      "Error/InvalidRequest",
			request:   &core.GenerationWatchRequest{},
			expectErr: core.ErrInvalidRequest,
		},
		{
			name:       "Error/InitialNotFound",
			request:    validRequest,
			callGenAI:  true,
			initialErr: status.Error(codes.NotFound, "not found"),
			expectErr:  core.ErrGenerationNotFound,
		},
		{
			name:       "Error/Initial",
			request:    validRequest,
			callGenAI:  true,
			initialErr: errFoo,
			expectErr:  errFoo,
		},
		{
			name:      "Error/Closed",
			request:   validRequest,
			callGenAI: true,
			responses: []*servicegenai.GenerationWatchResponse{{
				Generation: generationFixture(servicegenai.GenerationStatusPending, nil),
			}},
			expectErr: core.ErrGenerationWatchClosed,
		},
		{
			name:      "Error/ReceiveNotFound",
			request:   validRequest,
			callGenAI: true,
			streamErr: status.Error(codes.NotFound, "not found"),
			expectErr: core.ErrGenerationNotFound,
		},
		{
			name:      "Error/Receive",
			request:   validRequest,
			callGenAI: true,
			streamErr: errFoo,
			expectErr: errFoo,
		},
		{
			name:      "Error/MissingResponse",
			request:   validRequest,
			callGenAI: true,
			responses: []*servicegenai.GenerationWatchResponse{nil},
			expectErr: core.ErrGenerationResponseInvalid,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			engineVersionDao := coremocks.NewMockEngineVersionSelectDao(t)
			genai := servicegenaimocks.NewMockClient(t)

			if testCase.callGenAI {
				var stream grpc.ServerStreamingClient[servicegenai.GenerationWatchResponse]
				if testCase.initialErr == nil {
					stream = &generationWatchStream{
						responses: testCase.responses,
						err:       testCase.streamErr,
					}
				}

				genai.EXPECT().
					GenerationWatch(mock.Anything, &servicegenai.GenerationWatchRequest{
						Id:      testCase.request.ID.String(),
						OwnerId: testCase.request.Actor.UserID.String(),
					}).
					Return(stream, testCase.initialErr)
			}

			if testCase.engineResult != nil {
				engineVersionDao.EXPECT().
					Exec(mock.Anything, &dao.EngineVersionSelectRequest{ID: engineVersionID}).
					Return(testCase.engineResult, nil)
			}

			result, err := core.NewGenerationWatch(engineVersionDao, genai).
				Exec(t.Context(), testCase.request)

			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)
		})
	}
}
