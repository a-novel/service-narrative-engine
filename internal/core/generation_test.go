package core_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

func TestGenerationStatusTerminal(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		status core.GenerationStatus
		expect bool
	}{
		{status: core.GenerationStatusPending},
		{status: core.GenerationStatusRunning},
		{status: core.GenerationStatusSucceeded, expect: true},
		{status: core.GenerationStatusFailed, expect: true},
		{status: core.GenerationStatusAbandoned, expect: true},
		{status: core.GenerationStatusCancelled, expect: true},
		{status: core.GenerationStatus("unknown")},
	}

	for _, testCase := range testCases {
		require.Equal(t, testCase.expect, testCase.status.Terminal())
	}
}
