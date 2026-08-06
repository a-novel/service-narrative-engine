package lib_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/lib"
)

func TestParseRequiredRFC3339(t *testing.T) {
	t.Parallel()

	expect := time.Date(2026, time.August, 4, 12, 34, 56, 123, time.UTC)

	testCases := []struct {
		name string

		value string

		expect    time.Time
		expectErr bool
	}{
		{
			name:   "Success",
			value:  "2026-08-04T12:34:56.000000123Z",
			expect: expect,
		},
		{
			name:      "Error/Empty",
			expectErr: true,
		},
		{
			name:      "Error/Malformed",
			value:     "tomorrow",
			expectErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result, err := lib.ParseRequiredRFC3339("created_at", testCase.value)
			if testCase.expectErr {
				require.Error(t, err)
				require.Zero(t, result)

				return
			}

			require.NoError(t, err)
			require.Equal(t, testCase.expect, result)
		})
	}
}

func TestParseOptionalRFC3339(t *testing.T) {
	t.Parallel()

	expect := time.Date(2026, time.August, 4, 12, 34, 56, 123, time.UTC)

	testCases := []struct {
		name string

		value string

		expect    *time.Time
		expectErr bool
	}{
		{
			name: "Success/Absent",
		},
		{
			name:   "Success/Present",
			value:  "2026-08-04T12:34:56.000000123Z",
			expect: &expect,
		},
		{
			name:      "Error/Malformed",
			value:     "tomorrow",
			expectErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result, err := lib.ParseOptionalRFC3339("settled_at", testCase.value)
			if testCase.expectErr {
				require.Error(t, err)
				require.Nil(t, result)

				return
			}

			require.NoError(t, err)
			require.Equal(t, testCase.expect, result)
		})
	}
}
