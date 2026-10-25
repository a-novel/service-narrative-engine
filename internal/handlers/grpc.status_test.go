package handlers_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/a-novel/service-template/internal/handlers"
	templatev0 "github.com/a-novel/service-template/internal/handlers/protogen/anovel/template/v0"
	"github.com/a-novel/service-template/pkg/go"
)

func TestGrpcStatus(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		setup probeSetup

		expect       *templatev0.StatusResponse
		expectStatus codes.Code
	}{
		{
			name:  "Success",
			setup: probeHealthy,
			expect: &templatev0.StatusResponse{
				Postgres: &templatev0.DependencyHealth{
					Status: templatev0.DependencyStatus_DEPENDENCY_STATUS_UP,
				},
			},
			expectStatus: codes.OK,
		},
		{name: "Error/MissingPostgres", setup: probeWithoutPostgres, expectStatus: codes.Unavailable},
		{name: "Error/ClosedPostgres", setup: probeClosedPostgres, expectStatus: codes.Unavailable},
		{name: "Error/TransactionContext", setup: probeInTransaction, expectStatus: codes.Unavailable},
		{name: "Error/CancelledProbe", setup: probeCancelled, expectStatus: codes.Unavailable},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := newProbeContext(t, testCase.setup)

			response, err := handlers.NewGrpcStatus().Status(ctx, &templatev0.StatusRequest{})
			result, ok := status.FromError(err)
			require.True(t, ok)
			require.Equal(t, testCase.expectStatus, result.Code(), "got %s (%v)", result.Code(), err)
			require.Equal(t, testCase.expect, response)

			if testCase.expectStatus != codes.OK {
				// The message carries no raw dependency error.
				require.Equal(t, "service dependencies unavailable", result.Message())
			}
		})
	}

	t.Run("Transport/Unavailable", func(t *testing.T) {
		t.Parallel()

		listener := bufconn.Listen(1024 * 1024)
		server := grpc.NewServer()
		templatev0.RegisterStatusServiceServer(server, handlers.NewGrpcStatus())

		serveResult := make(chan error, 1)
		go func() { serveResult <- server.Serve(listener) }()

		t.Cleanup(func() {
			server.Stop()
			require.NoError(t, <-serveResult)
		})

		client, err := servicetemplate.NewClient(
			"passthrough:///health-test",
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
				return listener.DialContext(ctx)
			}),
		)
		require.NoError(t, err)

		defer client.Close()

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()

		response, err := client.Status(ctx, &servicetemplate.StatusRequest{})
		require.Nil(t, response)
		require.Equal(t, codes.Unavailable, status.Code(err))
		require.Equal(t, "service dependencies unavailable", status.Convert(err).Message())
	})
}
