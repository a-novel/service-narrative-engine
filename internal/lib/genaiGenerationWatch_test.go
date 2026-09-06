package lib_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel/service-narrative-engine/internal/lib"
)

func TestGenAIGenerationWatch(t *testing.T) {
	t.Parallel()

	running := genAIWireGeneration()
	running.Status = servicegenai.GenerationStatusRunning
	failed := genAIWireGeneration()
	failed.Status, failed.Error = servicegenai.GenerationStatusFailed, "private provider details"

	testCases := []struct {
		name                              string
		source                            *servicegenai.Generation
		initialErr, receiveErr, expectErr error
		nilStream, cancelParent           bool
	}{
		{name: "Success", source: running},
		{name: "Success/PrivateFailure", source: failed},
		{
			name:       "Error/SetupNotFound",
			initialErr: status.Error(codes.NotFound, "missing"), expectErr: lib.ErrGenerationNotFound,
		},
		{name: "Error/MissingStream", nilStream: true, expectErr: lib.ErrGenerationResponseInvalid},
		{
			name:       "Error/ReceiveNotFound",
			receiveErr: status.Error(codes.NotFound, "missing"), expectErr: lib.ErrGenerationNotFound,
		},
		{name: "Error/EOF", receiveErr: io.EOF, expectErr: io.EOF},
		{name: "Error/MissingGeneration", expectErr: lib.ErrGenerationResponseInvalid},
		{name: "Error/ParentCancelled", cancelParent: true, expectErr: context.Canceled},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := metadata.NewOutgoingContext(t.Context(), metadata.Pairs("x-request-id", t.Name()))

			ctx, cancel := context.WithTimeout(ctx, time.Minute)
			defer cancel()

			streamContexts := make(chan context.Context, 1)

			calls := 0
			gateway := lib.GenAIGenerationWatch(func(
				callCtx context.Context,
				request *servicegenai.GenerationWatchRequest,
				_ ...grpc.CallOption,
			) (grpc.ServerStreamingClient[servicegenai.GenerationWatchResponse], error) {
				calls++

				streamContexts <- callCtx

				assertGenAIContext(t, ctx, callCtx)
				require.Equal(t, genAIGenerationID.String(), request.GetId())
				require.Equal(t, genAIOwnerID.String(), request.GetOwnerId())

				if testCase.initialErr != nil || testCase.nilStream {
					return nil, testCase.initialErr
				}

				return &genAIWatchStream{receive: func() (*servicegenai.GenerationWatchResponse, error) {
					if testCase.cancelParent {
						<-callCtx.Done()

						return nil, callCtx.Err()
					}

					return &servicegenai.GenerationWatchResponse{Generation: testCase.source}, testCase.receiveErr
				}}, nil
			})

			stream, err := gateway.Exec(ctx, &lib.GenerationWatchGatewayRequest{ID: genAIGenerationID, OwnerID: genAIOwnerID})

			streamCtx := <-streamContexts

			if testCase.cancelParent {
				cancel()
			}

			if err == nil {
				var result *lib.Generation

				result, err = stream.Recv()
				if err == nil {
					require.Equal(t, genAIGenerationID, result.ID)
					require.Equal(t, testCase.source == failed, result.Failed)
					require.Empty(t, result.Output)
					require.Nil(t, result.SettledAt)
					require.NoError(t, streamCtx.Err())
					stream.Close()
				}
			}

			require.ErrorIs(t, err, testCase.expectErr)
			require.ErrorIs(t, streamCtx.Err(), context.Canceled)
			require.Equal(t, 1, calls)
		})
	}
}
