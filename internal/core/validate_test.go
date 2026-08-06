package core_test

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/core"
)

func TestValidateNotBlank(t *testing.T) {
	t.Parallel()

	type payload struct {
		Value string `validate:"notblank"`
	}

	testCases := []struct {
		name string

		value string

		expectValid bool
	}{
		{
			name:        "Success",
			value:       "A lighthouse keeper hears a second foghorn.",
			expectValid: true,
		},
		{
			name:        "Success/SurroundedByWhitespace",
			value:       "  speculative  ",
			expectValid: true,
		},
		{
			name:  "Error/Empty",
			value: "",
		},
		{
			name:  "Error/Spaces",
			value: "   ",
		},
		{
			name:  "Error/TabsAndNewlines",
			value: "\t\n\r",
		},
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	require.NoError(t, validate.RegisterValidation("notblank", core.ValidateNotBlank))

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := validate.Struct(payload{Value: testCase.value})

			if testCase.expectValid {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)
		})
	}
}

func TestValidateActor(t *testing.T) {
	t.Parallel()

	type payload struct {
		Actor core.Actor `validate:"actor"`
	}

	testCases := []struct {
		name string

		actor core.Actor

		expectValid bool
	}{
		{
			name:        "Success",
			actor:       core.Actor{UserID: uuid.MustParse("00000000-0000-0000-0000-000000000001")},
			expectValid: true,
		},
		{
			name:  "Error/Anonymous",
			actor: core.Actor{},
		},
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	require.NoError(t, validate.RegisterValidation("actor", core.ValidateActor))

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := validate.Struct(payload{Actor: testCase.actor})

			if testCase.expectValid {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)
		})
	}
}
