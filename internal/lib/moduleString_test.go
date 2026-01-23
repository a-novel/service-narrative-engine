package lib_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/lib"
)

func TestModuleStringRegexp(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		input   string
		matches bool
	}{
		{
			name:    "Valid",
			input:   "ab:cd@v1.0.0",
			matches: true,
		},
		{
			name:    "ValidLonger",
			input:   "namespace:module@v1.0.0",
			matches: true,
		},
		{
			name:    "ValidWithHyphens",
			input:   "my-namespace:my-module@v1.2.3",
			matches: true,
		},
		{
			name:    "ValidWithNumbers",
			input:   "namespace123:module456@v10.20.30",
			matches: true,
		},
		{
			name:    "ValidWithSingleCharPreversion",
			input:   "ab:cd@v1.0.0-a",
			matches: true,
		},
		{
			name:    "ValidWithMultiCharPreversion",
			input:   "ab:cd@v1.0.0-beta",
			matches: true,
		},
		{
			name:    "ValidWithAlphanumericPreversion",
			input:   "ab:cd@v1.0.0-alpha1",
			matches: true,
		},
		{
			name:    "ValidWithMultiplePreversions",
			input:   "ab:cd@v1.0.0-alpha-beta-rc1",
			matches: true,
		},
		{
			name:    "ValidNumericPreversions",
			input:   "my-namespace:my-module@v1.2.3-1-2-3",
			matches: true,
		},
		{
			name:    "ValidLeadingZerosInVersion",
			input:   "ab:cd@v01.0.0",
			matches: true,
		},
		{
			name:    "ValidSingleCharNamespace",
			input:   "a:cd@v1.0.0",
			matches: true,
		},
		{
			name:    "ValidSingleCharModule",
			input:   "ab:c@v1.0.0",
			matches: true,
		},
		{
			name:    "InvalidMissingNamespace",
			input:   ":cd@v1.0.0",
			matches: false,
		},
		{
			name:    "InvalidMissingModule",
			input:   "ab:@v1.0.0",
			matches: false,
		},
		{
			name:    "InvalidMissingVersion",
			input:   "ab:cd@v",
			matches: false,
		},
		{
			name:    "InvalidMissingVersionPrefix",
			input:   "ab:cd@1.0.0",
			matches: false,
		},
		{
			name:    "InvalidUppercaseNamespace",
			input:   "Ab:cd@v1.0.0",
			matches: false,
		},
		{
			name:    "InvalidUppercaseModule",
			input:   "ab:Cd@v1.0.0",
			matches: false,
		},
		{
			name:    "InvalidNamespaceStartsWithHyphen",
			input:   "-b:cd@v1.0.0",
			matches: false,
		},
		{
			name:    "InvalidModuleStartsWithHyphen",
			input:   "ab:-d@v1.0.0",
			matches: false,
		},
		{
			name:    "InvalidMissingColon",
			input:   "abcd@v1.0.0",
			matches: false,
		},
		{
			name:    "InvalidMissingAt",
			input:   "ab:cdv1.0.0",
			matches: false,
		},
		{
			name:    "InvalidPartialVersion",
			input:   "ab:cd@v1.0",
			matches: false,
		},
		{
			name:    "InvalidEmptyString",
			input:   "",
			matches: false,
		},
		{
			name:    "InvalidPreversionUppercase",
			input:   "ab:cd@v1.0.0-BETA",
			matches: false,
		},
		{
			name:    "InvalidPreversionMissingChar",
			input:   "ab:cd@v1.0.0-",
			matches: false,
		},
		{
			name:    "InvalidUnderscore",
			input:   "a_b:cd@v1.0.0",
			matches: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := lib.ModuleStringRegexp.MatchString(tc.input)
			require.Equal(t, tc.matches, result)
		})
	}
}

func TestDecodeModule(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		input  string
		expect lib.DecodedModule
	}{
		{
			name:  "SimpleModule",
			input: "ab:cd@v1.0.0",
			expect: lib.DecodedModule{
				Namespace:  "ab",
				Module:     "cd",
				Version:    "1.0.0",
				Preversion: "",
			},
		},
		{
			name:  "LongerModule",
			input: "namespace:module@v1.0.0",
			expect: lib.DecodedModule{
				Namespace:  "namespace",
				Module:     "module",
				Version:    "1.0.0",
				Preversion: "",
			},
		},
		{
			name:  "ModuleWithHyphens",
			input: "my-namespace:my-module@v1.2.3",
			expect: lib.DecodedModule{
				Namespace:  "my-namespace",
				Module:     "my-module",
				Version:    "1.2.3",
				Preversion: "",
			},
		},
		{
			name:  "ModuleWithNumbers",
			input: "namespace123:module456@v10.20.30",
			expect: lib.DecodedModule{
				Namespace:  "namespace123",
				Module:     "module456",
				Version:    "10.20.30",
				Preversion: "",
			},
		},
		{
			name:  "ModuleWithSingleCharPreversion",
			input: "ab:cd@v1.0.0-a",
			expect: lib.DecodedModule{
				Namespace:  "ab",
				Module:     "cd",
				Version:    "1.0.0",
				Preversion: "-a",
			},
		},
		{
			name:  "ModuleWithMultiCharPreversion",
			input: "ab:cd@v1.0.0-beta",
			expect: lib.DecodedModule{
				Namespace:  "ab",
				Module:     "cd",
				Version:    "1.0.0",
				Preversion: "-beta",
			},
		},
		{
			name:  "ModuleWithAlphanumericPreversion",
			input: "ab:cd@v1.0.0-alpha1",
			expect: lib.DecodedModule{
				Namespace:  "ab",
				Module:     "cd",
				Version:    "1.0.0",
				Preversion: "-alpha1",
			},
		},
		{
			name:  "ModuleWithMultiplePreversions",
			input: "ab:cd@v1.0.0-alpha-beta-rc1",
			expect: lib.DecodedModule{
				Namespace:  "ab",
				Module:     "cd",
				Version:    "1.0.0",
				Preversion: "-alpha-beta-rc1",
			},
		},
		{
			name:  "ComplexModule",
			input: "anovel-core:story-engine@v2.15.7-beta-1",
			expect: lib.DecodedModule{
				Namespace:  "anovel-core",
				Module:     "story-engine",
				Version:    "2.15.7",
				Preversion: "-beta-1",
			},
		},
		{
			name:  "LongNamespaceAndModule",
			input: "very-long-namespace-name:equally-long-module-name@v0.0.1",
			expect: lib.DecodedModule{
				Namespace:  "very-long-namespace-name",
				Module:     "equally-long-module-name",
				Version:    "0.0.1",
				Preversion: "",
			},
		},
		{
			name:  "NumericPreversion",
			input: "ab:cd@v1.0.0-1-2-3",
			expect: lib.DecodedModule{
				Namespace:  "ab",
				Module:     "cd",
				Version:    "1.0.0",
				Preversion: "-1-2-3",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := lib.DecodeModule(tc.input)
			require.Equal(t, tc.expect, result)
		})
	}
}

func TestDecodedModule_String(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		input  lib.DecodedModule
		expect string
	}{
		{
			name: "SimpleModule",
			input: lib.DecodedModule{
				Namespace:  "ab",
				Module:     "cd",
				Version:    "1.0.0",
				Preversion: "",
			},
			expect: "ab:cd@v1.0.0",
		},
		{
			name: "LongerModule",
			input: lib.DecodedModule{
				Namespace:  "namespace",
				Module:     "module",
				Version:    "1.0.0",
				Preversion: "",
			},
			expect: "namespace:module@v1.0.0",
		},
		{
			name: "ModuleWithHyphens",
			input: lib.DecodedModule{
				Namespace:  "my-namespace",
				Module:     "my-module",
				Version:    "1.2.3",
				Preversion: "",
			},
			expect: "my-namespace:my-module@v1.2.3",
		},
		{
			name: "ModuleWithNumbers",
			input: lib.DecodedModule{
				Namespace:  "namespace123",
				Module:     "module456",
				Version:    "10.20.30",
				Preversion: "",
			},
			expect: "namespace123:module456@v10.20.30",
		},
		{
			name: "ModuleWithSingleCharPreversion",
			input: lib.DecodedModule{
				Namespace:  "ab",
				Module:     "cd",
				Version:    "1.0.0",
				Preversion: "-a",
			},
			expect: "ab:cd@v1.0.0-a",
		},
		{
			name: "ModuleWithMultiCharPreversion",
			input: lib.DecodedModule{
				Namespace:  "ab",
				Module:     "cd",
				Version:    "1.0.0",
				Preversion: "-beta",
			},
			expect: "ab:cd@v1.0.0-beta",
		},
		{
			name: "ModuleWithAlphanumericPreversion",
			input: lib.DecodedModule{
				Namespace:  "ab",
				Module:     "cd",
				Version:    "1.0.0",
				Preversion: "-alpha1",
			},
			expect: "ab:cd@v1.0.0-alpha1",
		},
		{
			name: "ModuleWithMultiplePreversions",
			input: lib.DecodedModule{
				Namespace:  "ab",
				Module:     "cd",
				Version:    "1.0.0",
				Preversion: "-alpha-beta-rc1",
			},
			expect: "ab:cd@v1.0.0-alpha-beta-rc1",
		},
		{
			name: "ComplexModule",
			input: lib.DecodedModule{
				Namespace:  "anovel-core",
				Module:     "story-engine",
				Version:    "2.15.7",
				Preversion: "-beta-1",
			},
			expect: "anovel-core:story-engine@v2.15.7-beta-1",
		},
		{
			name: "LongNamespaceAndModule",
			input: lib.DecodedModule{
				Namespace:  "very-long-namespace-name",
				Module:     "equally-long-module-name",
				Version:    "0.0.1",
				Preversion: "",
			},
			expect: "very-long-namespace-name:equally-long-module-name@v0.0.1",
		},
		{
			name: "NumericPreversion",
			input: lib.DecodedModule{
				Namespace:  "ab",
				Module:     "cd",
				Version:    "1.0.0",
				Preversion: "-1-2-3",
			},
			expect: "ab:cd@v1.0.0-1-2-3",
		},
		{
			name: "NoVersion",
			input: lib.DecodedModule{
				Namespace:  "namespace",
				Module:     "module",
				Version:    "",
				Preversion: "",
			},
			expect: "namespace:module",
		},
		{
			name: "NoVersionWithHyphens",
			input: lib.DecodedModule{
				Namespace:  "my-namespace",
				Module:     "my-module",
				Version:    "",
				Preversion: "",
			},
			expect: "my-namespace:my-module",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := tc.input.String()
			require.Equal(t, tc.expect, result)
		})
	}
}
