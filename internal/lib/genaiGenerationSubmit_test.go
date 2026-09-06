package lib_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicegenai "github.com/a-novel/service-genai/pkg/go"
	servicegenaimocks "github.com/a-novel/service-genai/pkg/go/mocks"

	"github.com/a-novel/service-narrative-engine/internal/lib"
)

func TestGenAIGenerationSubmit(t *testing.T) {
	t.Parallel()

	errGateway := errors.New("generation gateway failure")
	request := &lib.GenerationSubmitGatewayRequest{
		OwnerID:        genAIOwnerID,
		Purpose:        "studio.generation",
		IdempotencyKey: "idempotency-key",
		Request:        []byte(`{"model":"gpt-test"}`),
		MaxAttempts:    2,
	}
	validResponse := &servicegenai.GenerationSubmitResponse{
		Generation: genAIWireGeneration(servicegenai.GenerationStatusPending, nil),
		Created:    true,
	}

	testCases := []struct {
		name string

		response *servicegenai.GenerationSubmitResponse
		err      error

		expect     *lib.GenerationSubmitGatewayResult
		expectErr  error
		expectCode codes.Code
	}{
		{
			name: "Success", response: validResponse,
			expect: &lib.GenerationSubmitGatewayResult{
				Generation: expectedGatewayGeneration(lib.GenerationStatusPending, ""),
				Created:    true,
			},
		},
		{
			name: "Error/Conflict", err: status.Error(codes.AlreadyExists, "conflict"),
			expectErr: lib.ErrGenerationConflict, expectCode: codes.AlreadyExists,
		},
		{
			name: "Error/PermissionDenied", err: status.Error(codes.PermissionDenied, "denied"),
			expectCode: codes.PermissionDenied,
		},
		{
			name: "Error/Unavailable", err: status.Error(codes.Unavailable, "unavailable"),
			expectCode: codes.Unavailable,
		},
		{name: "Error/Gateway", err: errGateway, expectErr: errGateway},
		{name: "Error/MissingResponse", expectErr: lib.ErrGenerationResponseInvalid},
		{
			name: "Error/MissingGeneration", response: &servicegenai.GenerationSubmitResponse{},
			expectErr: lib.ErrGenerationResponseInvalid,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			client := servicegenaimocks.NewMockClient(t)
			client.EXPECT().
				GenerationSubmit(mock.Anything, mock.Anything).
				RunAndReturn(func(
					_ context.Context,
					wireRequest *servicegenai.GenerationSubmitRequest,
					options ...grpc.CallOption,
				) (*servicegenai.GenerationSubmitResponse, error) {
					require.Equal(t, request.OwnerID.String(), wireRequest.GetOwnerId())
					require.Equal(t, request.Purpose, wireRequest.GetPurpose())
					require.Equal(t, request.IdempotencyKey, wireRequest.GetIdempotencyKey())
					require.Equal(t, request.Request, wireRequest.GetRequest())
					require.Equal(t, request.MaxAttempts, wireRequest.GetMaxAttempts())
					require.Empty(t, options)

					return testCase.response, testCase.err
				}).
				Once()

			result, err := lib.NewGenAIGenerationSubmit(client).Exec(t.Context(), request)

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
			client.AssertExpectations(t)
		})
	}
}
