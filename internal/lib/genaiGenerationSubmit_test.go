package lib_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel/service-narrative-engine/internal/lib"
)

func TestGenAIGenerationSubmit(t *testing.T) {
	t.Parallel()

	request := &lib.GenerationSubmitGatewayRequest{
		OwnerID: genAIOwnerID, Purpose: "studio.generation", IdempotencyKey: "retry-key",
		Request: []byte(`{"model":"gpt-test"}`), MaxAttempts: 2,
	}

	testCases := []struct {
		name      string
		response  *servicegenai.GenerationSubmitResponse
		err       error
		expectErr error
	}{
		{
			name:     "Success/Created",
			response: &servicegenai.GenerationSubmitResponse{Generation: genAIWireGeneration(), Created: true},
		},
		{name: "Success/Replay", response: &servicegenai.GenerationSubmitResponse{Generation: genAIWireGeneration()}},
		{name: "Error/Conflict", err: status.Error(codes.AlreadyExists, "conflict"), expectErr: lib.ErrGenerationConflict},
		{
			name:     "Error/MissingGeneration",
			response: &servicegenai.GenerationSubmitResponse{}, expectErr: lib.ErrGenerationResponseInvalid,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			calls := 0
			gateway := lib.GenAIGenerationSubmit(func(
				_ context.Context,
				wire *servicegenai.GenerationSubmitRequest,
				_ ...grpc.CallOption,
			) (*servicegenai.GenerationSubmitResponse, error) {
				calls++

				require.Equal(t, &servicegenai.GenerationSubmitRequest{
					OwnerId: request.OwnerID.String(), Purpose: request.Purpose,
					IdempotencyKey: request.IdempotencyKey, Request: request.Request, MaxAttempts: request.MaxAttempts,
				}, wire)

				return testCase.response, testCase.err
			})
			result, err := gateway.Exec(t.Context(), request)
			require.ErrorIs(t, err, testCase.expectErr)

			if err == nil {
				require.Equal(t, testCase.response.Created, result.Created)
				require.Equal(t, genAIGenerationID, result.Generation.ID)
			} else {
				require.Nil(t, result)
			}

			require.Equal(t, 1, calls)
		})
	}
}
