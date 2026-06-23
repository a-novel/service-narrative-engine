package handlers_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/handlers"
	"github.com/a-novel/service-narrative-engine/internal/handlers/protogen"
)

func TestStatus(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		expect       *protogen.StatusResponse
		expectStatus codes.Code
	}{
		{
			name: "Success",

			expect: &protogen.StatusResponse{
				Postgres: &protogen.DependencyHealth{
					Status: protogen.DependencyStatus_DEPENDENCY_STATUS_UP,
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			handler := handlers.NewGrpcStatus()

			ctx, err := postgres.NewContext(t.Context(), configtest.PostgresPreset)
			require.NoError(t, err)

			res, err := handler.Status(ctx, new(protogen.StatusRequest))
			resSt, ok := status.FromError(err)
			require.True(t, ok, resSt.Code().String())
			require.Equal(
				t,
				testCase.expectStatus, resSt.Code(),
				"expected status code %s, got %s (%v)", testCase.expectStatus, resSt.Code(), err,
			)
			require.Equal(t, testCase.expect, res)
		})
	}
}
