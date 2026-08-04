package core_test

import (
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
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

func TestGenerationTargetValidation(t *testing.T) {
	t.Parallel()

	validStep := core.GenerationTarget{
		Kind:            core.GenerationTargetKindStep,
		EngineVersionID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		StepKey:         "characters",
	}

	testCases := []struct {
		name string

		target core.GenerationTarget

		expectValid bool
	}{
		{
			name:        "Success/Idea",
			target:      core.GenerationTarget{Kind: core.GenerationTargetKindIdea},
			expectValid: true,
		},
		{
			name:        "Success/Manuscript",
			target:      core.GenerationTarget{Kind: core.GenerationTargetKindManuscript},
			expectValid: true,
		},
		{
			name:        "Success/Step",
			target:      validStep,
			expectValid: true,
		},
		{
			name:   "Error/UnknownKind",
			target: core.GenerationTarget{Kind: "unknown"},
		},
		{
			name: "Error/StaticEngineVersion",
			target: core.GenerationTarget{
				Kind:            core.GenerationTargetKindIdea,
				EngineVersionID: validStep.EngineVersionID,
			},
		},
		{
			name: "Error/StaticStepKey",
			target: core.GenerationTarget{
				Kind:    core.GenerationTargetKindManuscript,
				StepKey: validStep.StepKey,
			},
		},
		{
			name: "Error/StepEngineVersionMissing",
			target: core.GenerationTarget{
				Kind:    core.GenerationTargetKindStep,
				StepKey: validStep.StepKey,
			},
		},
		{
			name: "Error/StepKeyBlank",
			target: core.GenerationTarget{
				Kind:            core.GenerationTargetKindStep,
				EngineVersionID: validStep.EngineVersionID,
				StepKey:         "   ",
			},
		},
		{
			name: "Error/StepKeyTooLong",
			target: core.GenerationTarget{
				Kind:            core.GenerationTargetKindStep,
				EngineVersionID: validStep.EngineVersionID,
				StepKey:         strings.Repeat("a", 257),
			},
		},
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	require.NoError(t, validate.RegisterValidation("notblank", core.ValidateNotBlank))
	validate.RegisterStructValidation(core.ValidateGenerationTarget, core.GenerationTarget{})

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := validate.Struct(testCase.target)

			if testCase.expectValid {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)
		})
	}
}
