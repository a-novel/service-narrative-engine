package core_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
)

func TestGenerationGet(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	testCases := []struct {
		name string

		request *core.GenerationGetRequest

		genaiMock *daoMock[*servicegenai.Generation]

		expect *core.GenerationResult
		// expectFailure starts the reason of a failed result whose whole reason is long.
		expectFailure string
		expectErr     error
	}{
		{
			name: "Success",

			request: &core.GenerationGetRequest{OwnerID: ownerID, ID: genaiID},

			genaiMock: &daoMock[*servicegenai.Generation]{resp: genaiGeneration(
				servicegenai.GenerationStatusSucceeded,
				openaiResponse(core.GenerationKindSuggest, outputText(suggestOutput)),
			)},

			expect: &core.GenerationResult{
				ID: genaiID, Status: core.GenerationStatusSucceeded, Output: json.RawMessage(suggestOutput),
				Model: openaiModel,
			},
		},
		{
			name: "Success/Running",

			request: &core.GenerationGetRequest{OwnerID: ownerID, ID: genaiID},

			genaiMock: &daoMock[*servicegenai.Generation]{
				resp: genaiGeneration(servicegenai.GenerationStatusRunning, ""),
			},

			expect: &core.GenerationResult{ID: genaiID, Status: core.GenerationStatusRunning},
		},
		{
			name: "Success/Pending",

			request: &core.GenerationGetRequest{OwnerID: ownerID, ID: genaiID},

			genaiMock: &daoMock[*servicegenai.Generation]{
				resp: genaiGeneration(servicegenai.GenerationStatusPending, ""),
			},

			expect: &core.GenerationResult{ID: genaiID, Status: core.GenerationStatusPending},
		},
		{
			name: "Success/Failed",

			request: &core.GenerationGetRequest{OwnerID: ownerID, ID: genaiID},

			genaiMock: &daoMock[*servicegenai.Generation]{resp: func() *servicegenai.Generation {
				generation := genaiGeneration(servicegenai.GenerationStatusFailed, "")
				generation.Error = "provider unreachable"

				return generation
			}()},

			expect: &core.GenerationResult{
				ID: genaiID, Status: core.GenerationStatusFailed, Error: "provider unreachable",
			},
		},
		{
			name: "Success/Cancelled",

			request: &core.GenerationGetRequest{OwnerID: ownerID, ID: genaiID},

			genaiMock: &daoMock[*servicegenai.Generation]{
				resp: cancelledGeneration(),
			},

			expect: &core.GenerationResult{ID: genaiID, Status: core.GenerationStatusCancelled},
		},
		{
			// The model wrote its output in two parts.
			name: "Success/SplitText",

			request: &core.GenerationGetRequest{OwnerID: ownerID, ID: genaiID},

			genaiMock: &daoMock[*servicegenai.Generation]{resp: genaiGeneration(
				servicegenai.GenerationStatusSucceeded,
				openaiResponse(core.GenerationKindSuggest,
					outputText(suggestOutput[:20]), outputText(suggestOutput[20:])),
			)},

			expect: &core.GenerationResult{
				ID: genaiID, Status: core.GenerationStatusSucceeded, Output: json.RawMessage(suggestOutput),
				Model: openaiModel,
			},
		},
		{
			name: "Failed/Refused",

			request: &core.GenerationGetRequest{OwnerID: ownerID, ID: genaiID},

			genaiMock: &daoMock[*servicegenai.Generation]{resp: genaiGeneration(
				servicegenai.GenerationStatusSucceeded,
				openaiResponse(core.GenerationKindSuggest,
					map[string]string{"type": "refusal", "refusal": "I cannot help with that."}),
			)},

			expect: &core.GenerationResult{
				ID: genaiID, Status: core.GenerationStatusFailed, Error: "refused: I cannot help with that.",
			},
		},
		{
			name: "Failed/Truncated",

			request: &core.GenerationGetRequest{OwnerID: ownerID, ID: genaiID},

			genaiMock: &daoMock[*servicegenai.Generation]{resp: genaiGeneration(
				servicegenai.GenerationStatusSucceeded,
				`{"status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},"output":[]}`,
			)},

			expect: &core.GenerationResult{
				ID: genaiID, Status: core.GenerationStatusFailed, Error: "incomplete output: max_output_tokens",
			},
		},
		{
			name: "Failed/MalformedResponse",

			request: &core.GenerationGetRequest{OwnerID: ownerID, ID: genaiID},

			genaiMock: &daoMock[*servicegenai.Generation]{
				resp: genaiGeneration(servicegenai.GenerationStatusSucceeded, "not json"),
			},

			expect: &core.GenerationResult{
				ID: genaiID, Status: core.GenerationStatusFailed,
				Error: "malformed response: invalid character 'o' in literal null (expecting 'u')",
			},
		},
		{
			// A text suggestion for a manuscript target breaks the output schema.
			name: "Failed/MalformedOutput",

			request: &core.GenerationGetRequest{OwnerID: ownerID, ID: genaiID},

			genaiMock: &daoMock[*servicegenai.Generation]{resp: genaiGeneration(
				servicegenai.GenerationStatusSucceeded,
				openaiResponse(core.GenerationKindSuggest,
					outputText(`{"manuscript":[],"text":[{"target":"M1","note":"Rewrites.","text":""}]}`)),
			)},

			expectFailure: "malformed output: ",
		},
		{
			name: "Failed/UnknownFormat",

			request: &core.GenerationGetRequest{OwnerID: ownerID, ID: genaiID},

			genaiMock: &daoMock[*servicegenai.Generation]{resp: genaiGeneration(
				servicegenai.GenerationStatusSucceeded,
				openaiResponse("poem", outputText(suggestOutput)),
			)},

			expect: &core.GenerationResult{
				ID: genaiID, Status: core.GenerationStatusFailed, Error: "unknown output format: poem",
			},
		},
		{
			name: "Failed/NoResponse",

			request: &core.GenerationGetRequest{OwnerID: ownerID, ID: genaiID},

			genaiMock: &daoMock[*servicegenai.Generation]{
				resp: genaiGeneration(servicegenai.GenerationStatusSucceeded, ""),
			},

			expect: &core.GenerationResult{ID: genaiID, Status: core.GenerationStatusFailed, Error: "no response"},
		},
		{
			name: "Error/MissingID",

			request: &core.GenerationGetRequest{OwnerID: ownerID},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotFound",

			request: &core.GenerationGetRequest{OwnerID: ownerID, ID: genaiID},

			genaiMock: &daoMock[*servicegenai.Generation]{err: status.Error(codes.NotFound, "generation not found")},

			expectErr: core.ErrGenerationNotFound,
		},
		{
			name: "Error/Unavailable",

			request: &core.GenerationGetRequest{OwnerID: ownerID, ID: genaiID},

			genaiMock: &daoMock[*servicegenai.Generation]{err: status.Error(codes.Unavailable, "planned downtime")},

			expectErr: core.ErrGenaiUnavailable,
		},
		{
			name: "Error/Genai",

			request: &core.GenerationGetRequest{OwnerID: ownerID, ID: genaiID},

			genaiMock: &daoMock[*servicegenai.Generation]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockGenai := coremocks.NewMockGenerationGetServiceGenai(t)

			if testCase.genaiMock != nil {
				mockGenai.EXPECT().
					GenerationGet(mock.Anything, &servicegenai.GenerationGetRequest{
						Id: genaiID.String(), OwnerId: ownerID.String(),
					}).
					Return(&servicegenai.GenerationGetResponse{Generation: testCase.genaiMock.resp}, testCase.genaiMock.err)
			}

			service := core.NewGenerationGet(mockGenai)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)

			if testCase.expectFailure != "" {
				require.Equal(t, core.GenerationStatusFailed, result.Status)
				require.True(t, strings.HasPrefix(result.Error, testCase.expectFailure), result.Error)
				require.Empty(t, result.Output)
			} else {
				require.Equal(t, testCase.expect, result)
			}

			mockGenai.AssertExpectations(t)
		})
	}
}
