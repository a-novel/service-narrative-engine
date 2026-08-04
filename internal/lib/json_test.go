package lib_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/lib"
)

func TestDecodeJSONStrict(t *testing.T) {
	t.Parallel()

	type payload struct {
		Value string `json:"value"`
	}

	testCases := []struct {
		name string

		source []byte

		expect    payload
		expectErr error
	}{
		{
			name:   "Success",
			source: []byte(`{"value":"accepted"}`),
			expect: payload{Value: "accepted"},
		},
		{
			name:      "Error/Malformed",
			source:    []byte(`{`),
			expectErr: errors.New("decode JSON"),
		},
		{
			name:      "Error/UnknownField",
			source:    []byte(`{"value":"accepted","extra":true}`),
			expectErr: errors.New("decode JSON"),
		},
		{
			name:      "Error/MultipleValues",
			source:    []byte(`{"value":"accepted"} {}`),
			expectErr: lib.ErrJSONMultipleValues,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var result payload

			err := lib.DecodeJSONStrict(testCase.source, &result)

			if testCase.expectErr != nil && !errors.Is(testCase.expectErr, lib.ErrJSONMultipleValues) {
				require.ErrorContains(t, err, testCase.expectErr.Error())
			} else {
				require.ErrorIs(t, err, testCase.expectErr)
			}

			if testCase.expectErr == nil {
				require.Equal(t, testCase.expect, result)
			}
		})
	}
}
