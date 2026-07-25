package config_test

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	servicejobs "github.com/a-novel/service-jobs/pkg/go"

	"github.com/a-novel/service-narrative-engine/internal/config"
)

func TestDependencies(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
	}{
		{name: "Success/ServiceJobs"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg := config.AppPresetDefault.Dependencies
			options, err := cfg.ServiceJobsCredentials.Options(t.Context())
			require.NoError(t, err)

			client, err := servicejobs.NewClient(
				fmt.Sprintf("%s:%d", cfg.ServiceJobsHost, cfg.ServiceJobsPort),
				options...,
			)
			require.NoError(t, err)
			t.Cleanup(client.Close)

			ownerID := uuid.NewString()
			idempotencyKey := uuid.NewString()
			enqueued, err := client.JobEnqueue(t.Context(), &servicejobs.JobEnqueueRequest{
				Kind:           "connectivity",
				Payload:        []byte(`{"source":"service-narrative-engine"}`),
				OwnerId:        ownerID,
				IdempotencyKey: &idempotencyKey,
				MaxAttempts:    1,
			})
			require.NoError(t, err)
			require.True(t, enqueued.GetCreated())

			got, err := client.JobGet(t.Context(), &servicejobs.JobGetRequest{
				Id:      enqueued.GetJob().GetId(),
				OwnerId: ownerID,
			})
			require.NoError(t, err)
			require.Equal(t, enqueued.GetJob().GetId(), got.GetJob().GetId())
		})
	}
}
