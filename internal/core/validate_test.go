package core_test

import (
	"testing"

	"github.com/go-playground/validator/v10"
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

func TestValidateMaxBytes(t *testing.T) {
	t.Parallel()

	type payload struct {
		Value string `validate:"maxbytes=4"`
	}

	testCases := []struct {
		name string

		value string

		expectValid bool
	}{
		{
			name:        "Success/Empty",
			value:       "",
			expectValid: true,
		},
		{
			name:        "Success/AtLimit",
			value:       "éé",
			expectValid: true,
		},
		{
			name:  "Error/OverLimit",
			value: "ééa",
		},
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	require.NoError(t, validate.RegisterValidation("maxbytes", core.ValidateMaxBytes))

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

func TestValidateMaxBytesInvalidParameter(t *testing.T) {
	t.Parallel()

	type payload struct {
		Value string `validate:"maxbytes=invalid"`
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	require.NoError(t, validate.RegisterValidation("maxbytes", core.ValidateMaxBytes))
	require.Error(t, validate.Struct(payload{Value: "value"}))
}
