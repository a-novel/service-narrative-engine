package core_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
)

func TestGenerationCancel(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	testCases := []struct {
		name string

		request *core.GenerationCancelRequest

		genaiMock *daoMock[*servicegenai.Generation]

		expect    *core.GenerationResult
		expectErr error
	}{
		{
			name: "Success",

			request: &core.GenerationCancelRequest{OwnerID: ownerID, ID: genaiID},

			genaiMock: &daoMock[*servicegenai.Generation]{
				resp: genaiGeneration(servicegenai.GenerationStatusCancelled, ""),
			},

			expect: &core.GenerationResult{ID: genaiID, Status: core.GenerationStatusCancelled},
		},
		{
			// The call settled before the cancel reached it, so its paid output comes back.
			name: "Success/SettledFirst",

			request: &core.GenerationCancelRequest{OwnerID: ownerID, ID: genaiID},

			genaiMock: &daoMock[*servicegenai.Generation]{resp: genaiGeneration(
				servicegenai.GenerationStatusSucceeded,
				openaiResponse(core.GenerationKindSuggest, outputText(suggestOutput)),
			)},

			expect: &core.GenerationResult{
				ID: genaiID, Status: core.GenerationStatusSucceeded, Output: json.RawMessage(suggestOutput),
			},
		},
		{
			name: "Error/MissingOwner",

			request: &core.GenerationCancelRequest{ID: genaiID},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotFound",

			request: &core.GenerationCancelRequest{OwnerID: ownerID, ID: genaiID},

			genaiMock: &daoMock[*servicegenai.Generation]{err: status.Error(codes.NotFound, "generation not found")},

			expectErr: core.ErrGenerationNotFound,
		},
		{
			name: "Error/Genai",

			request: &core.GenerationCancelRequest{OwnerID: ownerID, ID: genaiID},

			genaiMock: &daoMock[*servicegenai.Generation]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockGenai := coremocks.NewMockGenerationCancelServiceGenai(t)

			if testCase.genaiMock != nil {
				mockGenai.EXPECT().
					GenerationCancel(mock.Anything, &servicegenai.GenerationCancelRequest{
						Id: genaiID.String(), OwnerId: ownerID.String(),
					}).
					Return(
						&servicegenai.GenerationCancelResponse{Generation: testCase.genaiMock.resp},
						testCase.genaiMock.err,
					)
			}

			service := core.NewGenerationCancel(mockGenai)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockGenai.AssertExpectations(t)
		})
	}
}
