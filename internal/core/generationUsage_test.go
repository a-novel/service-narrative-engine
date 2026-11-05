package core_test

import (
	"errors"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	servicegenai "github.com/a-novel/service-genai/pkg/go"

	"github.com/a-novel/service-narrative-engine/internal/config"
	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
)

func TestGenerationUsage(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	expectUserCaps := &core.GenerationCaps{
		Concurrency: lo.ToPtr[int32](2),
		Windows: []*core.GenerationCapWindow{
			{Duration: 5 * time.Hour, InputTokens: lo.ToPtr[int64](2_000_000), OutputTokens: lo.ToPtr[int64](200_000)},
			{
				Duration:    720 * time.Hour,
				InputTokens: lo.ToPtr[int64](50_000_000), OutputTokens: lo.ToPtr[int64](5_000_000),
			},
		},
	}

	testCases := []struct {
		name string

		request *core.GenerationUsageRequest

		// genaiMock answers the usage summary genai receives, which sums the windows.
		genaiMock *daoMock[*servicegenai.UsageSummaryResponse]
		windows   []time.Duration

		expect    *core.GenerationUsageSummary
		expectErr error
	}{
		{
			name: "Success",

			request: &core.GenerationUsageRequest{OwnerID: ownerID, Roles: userRoles},

			genaiMock: &daoMock[*servicegenai.UsageSummaryResponse]{resp: &servicegenai.UsageSummaryResponse{
				Live: 1,
				Windows: []*servicegenai.UsageSummaryWindow{
					{Duration: durationpb.New(5 * time.Hour), InputTokens: 1200, OutputTokens: 300},
					{Duration: durationpb.New(720 * time.Hour), InputTokens: 9000, OutputTokens: 2000},
				},
			}},
			windows: []time.Duration{5 * time.Hour, 720 * time.Hour},

			expect: &core.GenerationUsageSummary{
				Caps:     expectUserCaps,
				InFlight: 1,
				Windows: []*core.GenerationUsageWindow{
					{Duration: 5 * time.Hour, InputTokens: 1200, OutputTokens: 300},
					{Duration: 720 * time.Hour, InputTokens: 9000, OutputTokens: 2000},
				},
			},
		},
		{
			// An uncapped caller has no window to sum, and reads only their calls in flight.
			name: "Success/Uncapped",

			request: &core.GenerationUsageRequest{OwnerID: ownerID, Roles: []string{"auth:admin"}},

			genaiMock: &daoMock[*servicegenai.UsageSummaryResponse]{resp: &servicegenai.UsageSummaryResponse{Live: 3}},

			expect: &core.GenerationUsageSummary{
				Caps: &core.GenerationCaps{}, InFlight: 3, Windows: []*core.GenerationUsageWindow{},
			},
		},
		{
			name: "Error/MissingOwner",

			request: &core.GenerationUsageRequest{Roles: userRoles},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NoCaps",

			request: &core.GenerationUsageRequest{OwnerID: ownerID, Roles: []string{"auth:anon"}},

			expectErr: core.ErrNoGenerationCaps,
		},
		{
			name: "Error/Unavailable",

			request: &core.GenerationUsageRequest{OwnerID: ownerID, Roles: userRoles},

			genaiMock: &daoMock[*servicegenai.UsageSummaryResponse]{
				err: status.Error(codes.Unavailable, "planned downtime"),
			},
			windows: []time.Duration{5 * time.Hour, 720 * time.Hour},

			expectErr: core.ErrGenaiUnavailable,
		},
		{
			name: "Error/Genai",

			request: &core.GenerationUsageRequest{OwnerID: ownerID, Roles: userRoles},

			genaiMock: &daoMock[*servicegenai.UsageSummaryResponse]{err: errFoo},
			windows:   []time.Duration{5 * time.Hour, 720 * time.Hour},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockGenai := coremocks.NewMockGenerationUsageServiceGenai(t)

			if testCase.genaiMock != nil {
				expectRequest := &servicegenai.UsageSummaryRequest{
					OwnerId: ownerID.String(), Purpose: "studio.generation",
					Windows: lo.Map(testCase.windows, func(window time.Duration, _ int) *durationpb.Duration {
						return durationpb.New(window)
					}),
				}

				mockGenai.EXPECT().
					UsageSummary(mock.Anything, mock.MatchedBy(func(request *servicegenai.UsageSummaryRequest) bool {
						return proto.Equal(request, expectRequest)
					})).
					Return(testCase.genaiMock.resp, testCase.genaiMock.err)
			}

			caps, err := core.NewGenerationCapsPolicy(config.PermissionsConfigDefault, config.CapsConfigDefault)
			require.NoError(t, err)

			service := core.NewGenerationUsage(mockGenai, caps)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)

			mockGenai.AssertExpectations(t)
		})
	}
}
