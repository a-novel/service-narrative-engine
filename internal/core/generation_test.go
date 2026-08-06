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
		expect bool
	}{
		{name: "Pending", status: core.GenerationStatusPending},
		{name: "Running", status: core.GenerationStatusRunning},
		{name: "Succeeded", status: core.GenerationStatusSucceeded, expect: true},
		{name: "Failed", status: core.GenerationStatusFailed, expect: true},
		{name: "Abandoned", status: core.GenerationStatusAbandoned, expect: true},
		{name: "Cancelled", status: core.GenerationStatusCancelled, expect: true},
		{name: "Unknown", status: core.GenerationStatus("unknown")},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, testCase.expect, testCase.status.Terminal())
		})
	}
}
