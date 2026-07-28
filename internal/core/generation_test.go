package core_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

func TestGenerationStatus(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		status core.GenerationStatus

		expectTerminal bool
	}{
		{
			name:   "Pending",
			status: core.GenerationStatusPending,
		},
		{
			name:   "Running",
			status: core.GenerationStatusRunning,
		},
		{
			name:           "Succeeded",
			status:         core.GenerationStatusSucceeded,
			expectTerminal: true,
		},
		{
			name:           "Failed",
			status:         core.GenerationStatusFailed,
			expectTerminal: true,
		},
		{
			name:           "Abandoned",
			status:         core.GenerationStatusAbandoned,
			expectTerminal: true,
		},
		{
			name:           "Cancelled",
			status:         core.GenerationStatusCancelled,
			expectTerminal: true,
		},
		{
			// An unmapped status is not terminal, so a watch keeps waiting
			// rather than settling on a state it cannot interpret.
			name:   "Unknown",
			status: core.GenerationStatus("unknown"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, testCase.expectTerminal, testCase.status.Terminal())
		})
	}
}
