package lib_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel/service-narrative-engine/internal/lib"
)

func TestGenAIGenerationGet(t *testing.T) {
	t.Parallel()

	completed := genAIWireGeneration()
	completed.Status = servicegenai.GenerationStatusSucceeded
	completed.Output = []byte(`{"output":[{"content":[{"type":"output_text","text":"{}"}]}]}`)
	completed.SettledAt = genAICreatedAt.Add(2 * time.Second).Format(time.RFC3339Nano)
	completed.ExpiresAt = genAICreatedAt.Add(time.Hour).Format(time.RFC3339Nano)
	settledAt, expiresAt := genAICreatedAt.Add(2*time.Second), genAICreatedAt.Add(time.Hour)
	expect := &lib.Generation{
		ID: genAIGenerationID, OwnerID: genAIOwnerID, Purpose: "studio.generation",
		Status: lib.GenerationStatusSucceeded, Attempt: 1, MaxAttempts: 2, Output: "{}",
		CreatedAt: genAICreatedAt, UpdatedAt: genAICreatedAt.Add(time.Second),
		SettledAt: &settledAt, ExpiresAt: &expiresAt,
	}

	testCases := []struct {
		name      string
		change    func(*servicegenai.Generation)
		rpcErr    error
		expectErr error
	}{
		{name: "Success"},
		{name: "Error/NotFound", rpcErr: status.Error(codes.NotFound, "missing"), expectErr: lib.ErrGenerationNotFound},
		{name: "Error/PermissionDenied", rpcErr: status.Error(codes.PermissionDenied, "denied")},
		{name: "Error/Unavailable", rpcErr: status.Error(codes.Unavailable, "unavailable")},
		{name: "Error/Cancelled", rpcErr: status.Error(codes.Canceled, "cancelled")},
		{name: "Error/Deadline", rpcErr: status.Error(codes.DeadlineExceeded, "deadline")},
		{
			name:   "Error/MalformedID",
			change: func(g *servicegenai.Generation) { g.Id = "invalid" }, expectErr: lib.ErrGenerationResponseInvalid,
		},
		{
			name:   "Error/MalformedTime",
			change: func(g *servicegenai.Generation) { g.CreatedAt = "invalid" }, expectErr: lib.ErrGenerationResponseInvalid,
		},
		{
			name:   "Error/UnknownStatus",
			change: func(g *servicegenai.Generation) { g.Status = 99 }, expectErr: lib.ErrGenerationStatusUnknown,
		},
		{
			name:   "Error/Output",
			change: func(g *servicegenai.Generation) { g.Output = []byte("{") }, expectErr: lib.ErrGenerationOutputInvalid,
		},
		{name: "Error/Refusal", change: func(g *servicegenai.Generation) {
			g.Output = []byte(`{"output":[{"content":[{"type":"refusal","refusal":"private detail"}]}]}`)
		}, expectErr: lib.ErrGenerationRefused},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := proto.CloneOf(completed)
			if testCase.change != nil {
				testCase.change(source)
			}

			ctx := metadata.NewOutgoingContext(t.Context(), metadata.Pairs("x-request-id", t.Name()))

			ctx, cancel := context.WithTimeout(ctx, time.Minute)
			defer cancel()

			calls := 0
			gateway := lib.GenAIGenerationGet(func(
				callCtx context.Context,
				request *servicegenai.GenerationGetRequest,
				_ ...grpc.CallOption,
			) (*servicegenai.GenerationGetResponse, error) {
				calls++

				assertGenAIContext(t, ctx, callCtx)
				require.Equal(t, genAIGenerationID.String(), request.GetId())
				require.Equal(t, genAIOwnerID.String(), request.GetOwnerId())

				return &servicegenai.GenerationGetResponse{Generation: source}, testCase.rpcErr
			})
			result, err := gateway.Exec(ctx, &lib.GenerationGetGatewayRequest{ID: genAIGenerationID, OwnerID: genAIOwnerID})

			expectErr := testCase.expectErr
			if expectErr == nil {
				expectErr = testCase.rpcErr
			}

			require.ErrorIs(t, err, expectErr)

			if testCase.rpcErr != nil {
				require.Equal(t, status.Code(testCase.rpcErr), status.Code(err))
			}

			if expectErr == nil {
				require.Equal(t, expect, result)
			} else {
				require.Nil(t, result)
			}

			require.Equal(t, 1, calls)
		})
	}
}
