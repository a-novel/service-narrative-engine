package lib_test

import (
	"context"
	"errors"
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

func TestGenAIGenerationGet(t *testing.T) {
	t.Parallel()

	errGateway := errors.New("generation gateway failure")
	proposal := map[string]any{"shape": "belongs to the client"}
	proposalJSON := `{"shape":"belongs to the client"}`
	validResponse := &servicegenai.GenerationGetResponse{Generation: genAIWireGeneration(
		servicegenai.GenerationStatusSucceeded,
		genAIResponsesOutput(t, proposal),
	)}
	failed := genAIWireGeneration(servicegenai.GenerationStatusFailed, nil)
	failed.Error = "private provider details"
	expectFailed := expectedGatewayGeneration(lib.GenerationStatusFailed, "")
	expectFailed.Failed = true
	invalidID := genAIWireGeneration(servicegenai.GenerationStatusPending, nil)
	invalidID.Id = "not-a-uuid"
	invalidOwnerID := genAIWireGeneration(servicegenai.GenerationStatusPending, nil)
	invalidOwnerID.OwnerId = "not-a-uuid"
	unknownStatus := genAIWireGeneration(servicegenai.GenerationStatus(99), nil)
	invalidCreatedAt := genAIWireGeneration(servicegenai.GenerationStatusPending, nil)
	invalidCreatedAt.CreatedAt = "not-a-time"
	invalidSettledAt := genAIWireGeneration(servicegenai.GenerationStatusSucceeded, nil)
	invalidSettledAt.SettledAt = "not-a-time"

	testCases := []struct {
		name string

		response *servicegenai.GenerationGetResponse
		err      error

		expect     *lib.Generation
		expectErr  error
		expectCode codes.Code
	}{
		{
			name: "Success/Conversion", response: validResponse,
			expect: expectedGatewayGeneration(lib.GenerationStatusSucceeded, proposalJSON),
		},
		{
			name:     "Success/ProviderFailureIsOpaque",
			response: &servicegenai.GenerationGetResponse{Generation: failed},
			expect:   expectFailed,
		},
		{
			name: "Error/NotFound", err: status.Error(codes.NotFound, "not found"),
			expectErr: lib.ErrGenerationNotFound, expectCode: codes.NotFound,
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
			name: "Error/MissingGeneration", response: &servicegenai.GenerationGetResponse{},
			expectErr: lib.ErrGenerationResponseInvalid,
		},
		{
			name:      "Error/InvalidID",
			response:  &servicegenai.GenerationGetResponse{Generation: invalidID},
			expectErr: lib.ErrGenerationResponseInvalid,
		},
		{
			name:      "Error/InvalidOwnerID",
			response:  &servicegenai.GenerationGetResponse{Generation: invalidOwnerID},
			expectErr: lib.ErrGenerationResponseInvalid,
		},
		{
			name:      "Error/UnknownStatus",
			response:  &servicegenai.GenerationGetResponse{Generation: unknownStatus},
			expectErr: lib.ErrGenerationStatusUnknown,
		},
		{
			name:      "Error/InvalidCreatedAt",
			response:  &servicegenai.GenerationGetResponse{Generation: invalidCreatedAt},
			expectErr: lib.ErrGenerationResponseInvalid,
		},
		{
			name:      "Error/InvalidSettledAt",
			response:  &servicegenai.GenerationGetResponse{Generation: invalidSettledAt},
			expectErr: lib.ErrGenerationResponseInvalid,
		},
		{
			name: "Error/InvalidOutput",
			response: &servicegenai.GenerationGetResponse{Generation: genAIWireGeneration(
				servicegenai.GenerationStatusSucceeded,
				[]byte("{"),
			)},
			expectErr: lib.ErrGenerationOutputInvalid,
		},
		{
			name: "Error/Refused",
			response: &servicegenai.GenerationGetResponse{Generation: genAIWireGeneration(
				servicegenai.GenerationStatusSucceeded,
				genAIResponsesRefusal(t),
			)},
			expectErr: lib.ErrGenerationRefused,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			client := servicegenaimocks.NewMockClient(t)
			ctx := metadata.NewOutgoingContext(t.Context(), metadata.Pairs("x-request-id", t.Name()))

			ctx, cancel := context.WithTimeout(ctx, time.Minute)
			defer cancel()

			client.EXPECT().
				GenerationGet(mock.Anything, &servicegenai.GenerationGetRequest{
					Id: genAIGenerationID.String(), OwnerId: genAIOwnerID.String(),
				}).
				RunAndReturn(func(
					callCtx context.Context,
					_ *servicegenai.GenerationGetRequest,
					options ...grpc.CallOption,
				) (*servicegenai.GenerationGetResponse, error) {
					metadataValue, ok := metadata.FromOutgoingContext(callCtx)
					require.True(t, ok)
					require.Equal(t, []string{t.Name()}, metadataValue.Get("x-request-id"))

					_, hasDeadline := callCtx.Deadline()
					require.True(t, hasDeadline)
					require.Empty(t, options)

					return testCase.response, testCase.err
				}).
				Once()

			result, err := lib.NewGenAIGenerationGet(client).Exec(ctx, &lib.GenerationGetGatewayRequest{
				ID: genAIGenerationID, OwnerID: genAIOwnerID,
			})

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
