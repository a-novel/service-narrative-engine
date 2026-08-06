package lib_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/lib"
)

func TestValidateJSON(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		source   []byte
		maxBytes int

		expectErr error
	}{
		{name: "Success/Object", source: []byte(`{"freeform":true}`), maxBytes: 17},
		{name: "Success/Array", source: []byte(`[1,"two"]`), maxBytes: 9},
		{name: "Success/String", source: []byte(`"freeform"`), maxBytes: 10},
		{name: "Success/Number", source: []byte(`42`), maxBytes: 2},
		{name: "Success/Boolean", source: []byte(`true`), maxBytes: 4},
		{name: "Success/Null", source: []byte(`null`), maxBytes: 4},
		{name: "Success/Unlimited", source: []byte(`{"large":"accepted"}`)},
		{name: "Error/Empty", maxBytes: 1, expectErr: lib.ErrJSONEmpty},
		{name: "Error/Invalid", source: []byte(`{`), maxBytes: 1, expectErr: lib.ErrJSONInvalid},
		{
			name: "Error/InvalidUTF8", source: []byte{'"', 0xff, '"'}, maxBytes: 3,
			expectErr: lib.ErrJSONInvalid,
		},
		{name: "Error/TooLarge", source: []byte(`true`), maxBytes: 3, expectErr: lib.ErrJSONTooLarge},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := lib.ValidateJSON(testCase.source, testCase.maxBytes)
			require.ErrorIs(t, err, testCase.expectErr)
		})
	}
}
