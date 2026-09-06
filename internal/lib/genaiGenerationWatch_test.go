package lib_test

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	servicegenai "github.com/a-novel/service-genai/pkg/go"
	servicegenaimocks "github.com/a-novel/service-genai/pkg/go/mocks"

	"github.com/a-novel/service-narrative-engine/internal/lib"
)

func TestGenAIGenerationWatch(t *testing.T) {
	t.Parallel()

	errStream := errors.New("stream failure")
	validResponse := &servicegenai.GenerationWatchResponse{
		Generation: genAIWireGeneration(servicegenai.GenerationStatusRunning, nil),
	}

	testCases := []struct {
		name string

		responses  []*servicegenai.GenerationWatchResponse
		streamErr  error
		initialErr error
		nilStream  bool

		expect     *lib.Generation
		expectErr  error
		expectCode codes.Code
	}{
		{
			name: "Success/Conversion", responses: []*servicegenai.GenerationWatchResponse{validResponse},
			expect: expectedGatewayGeneration(lib.GenerationStatusRunning, ""),
		},
		{
			name: "Error/InitialNotFound", initialErr: status.Error(codes.NotFound, "not found"),
			expectErr: lib.ErrGenerationNotFound, expectCode: codes.NotFound,
		},
		{
			name: "Error/InitialUnavailable", initialErr: status.Error(codes.Unavailable, "unavailable"),
			expectCode: codes.Unavailable,
		},
		{
			name: "Error/MissingStream", nilStream: true,
			expectErr: lib.ErrGenerationResponseInvalid,
		},
		{
			name: "Error/ReceiveNotFound", streamErr: status.Error(codes.NotFound, "not found"),
			expectErr: lib.ErrGenerationNotFound, expectCode: codes.NotFound,
		},
		{
			name: "Error/ReceiveUnavailable", streamErr: status.Error(codes.Unavailable, "unavailable"),
			expectCode: codes.Unavailable,
		},
		{name: "Error/Receive", streamErr: errStream, expectErr: errStream},
		{name: "Error/EOF", streamErr: io.EOF, expectErr: io.EOF},
		{
			name: "Error/MissingResponse", responses: []*servicegenai.GenerationWatchResponse{nil},
			expectErr: lib.ErrGenerationResponseInvalid,
		},
		{
			name:      "Error/MissingGeneration",
			responses: []*servicegenai.GenerationWatchResponse{{}},
			expectErr: lib.ErrGenerationResponseInvalid,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			client := servicegenaimocks.NewMockClient(t)

			var wireStream grpc.ServerStreamingClient[servicegenai.GenerationWatchResponse]
			if testCase.initialErr == nil && !testCase.nilStream {
				wireStream = &genAIWatchStream{
					responses: testCase.responses,
					err:       testCase.streamErr,
				}
			}

			ctx := metadata.NewOutgoingContext(t.Context(), metadata.Pairs("x-request-id", t.Name()))

			ctx, cancel := context.WithTimeout(ctx, time.Minute)
			defer cancel()

			streamCtxChannel := make(chan context.Context, 1)

			client.EXPECT().
				GenerationWatch(mock.Anything, &servicegenai.GenerationWatchRequest{
					Id: genAIGenerationID.String(), OwnerId: genAIOwnerID.String(),
				}).
				RunAndReturn(func(
					callCtx context.Context,
					_ *servicegenai.GenerationWatchRequest,
					options ...grpc.CallOption,
				) (grpc.ServerStreamingClient[servicegenai.GenerationWatchResponse], error) {
					streamCtxChannel <- callCtx

					metadataValue, ok := metadata.FromOutgoingContext(callCtx)
					require.True(t, ok)
					require.Equal(t, []string{t.Name()}, metadataValue.Get("x-request-id"))

					_, hasDeadline := callCtx.Deadline()
					require.True(t, hasDeadline)
					require.Empty(t, options)

					return wireStream, testCase.initialErr
				}).
				Once()

			stream, err := lib.NewGenAIGenerationWatch(client).Exec(
				ctx,
				&lib.GenerationWatchGatewayRequest{ID: genAIGenerationID, OwnerID: genAIOwnerID},
			)

			var result *lib.Generation
			if err == nil {
				result, err = stream.Recv()
				if err == nil {
					stream.Close()
				}
			}

			if testCase.expectErr != nil {
				require.ErrorIs(t, err, testCase.expectErr)
			} else if testCase.expectCode == codes.OK {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}

			if testCase.expectCode != codes.OK {
				require.Equal(t, testCase.expectCode, status.Code(err))
			}

			require.Equal(t, testCase.expect, result)

			streamCtx := <-streamCtxChannel
			require.NotNil(t, streamCtx)

			select {
			case <-streamCtx.Done():
			case <-time.After(time.Second):
				t.Fatal("generation watch stream context was not cancelled")
			}

			client.AssertExpectations(t)
		})
	}
}
