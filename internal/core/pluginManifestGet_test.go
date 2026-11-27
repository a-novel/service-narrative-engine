package core_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/v2/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/v2/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
)

func TestPluginManifestGet(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	composer := dao.PluginVersionKey{Plugin: "scenes-composer", Major: 1, Minor: 10}
	composerDev := dao.PluginVersionKey{Plugin: "scenes-composer", Major: 2, Prerelease: "dev.1"}

	manifest := func(key dao.PluginVersionKey) *dao.PluginManifest {
		return &dao.PluginManifest{
			PluginVersionKey: key, Schema: []byte(`{"type": "object"}`), Exports: []byte(`{}`), HasMigration: true,
			SchemaChecksum: "schema checksum", UIChecksum: "ui checksum", PublishedAt: createdAt,
		}
	}

	expect := func(version string) *core.PluginManifest {
		return &core.PluginManifest{
			Plugin: "scenes-composer", Version: version, Schema: []byte(`{"type": "object"}`), Exports: []byte(`{}`),
			HasMigration: true, SchemaChecksum: "schema checksum", UIChecksum: "ui checksum", PublishedAt: createdAt,
		}
	}

	// A version published with an API key carries its proof, and the ID of that key.
	signedManifest := manifest(composer)
	signedManifest.Proof, signedManifest.APIKeyID = new("header.claims.signature"), new("publisher-key")

	expectSigned := expect("1.10.0")
	expectSigned.Proof, expectSigned.APIKeyID = new("header.claims.signature"), new("publisher-key")

	type testCase struct {
		name string

		request *core.PluginManifestGetRequest

		// daoRequest is the read the service asks for. The DAO is not called when it is nil.
		daoRequest *dao.PluginManifestSelectRequest
		daoMock    daoMock[*dao.PluginManifest]

		expect    *core.PluginManifest
		expectErr error
	}

	// invalid returns a case for a version the registry cannot hold, which no DAO reads.
	invalid := func(name, version string) testCase {
		return testCase{
			name:      "Error/InvalidVersion/" + name,
			request:   &core.PluginManifestGetRequest{Plugin: "scenes-composer", Version: version},
			expectErr: core.ErrInvalidRequest,
		}
	}

	// invalidPlugin returns a case for a name that is no DNS label, which no DAO reads.
	invalidPlugin := func(name, plugin string) testCase {
		return testCase{
			name:      "Error/InvalidPlugin/" + name,
			request:   &core.PluginManifestGetRequest{Plugin: plugin, Version: "1.10.0"},
			expectErr: core.ErrInvalidRequest,
		}
	}

	testCases := []testCase{
		{
			name: "Success",

			request: &core.PluginManifestGetRequest{Plugin: "scenes-composer", Version: "1.10.0"},

			daoRequest: &dao.PluginManifestSelectRequest{PluginVersionKey: composer},
			daoMock:    daoMock[*dao.PluginManifest]{resp: manifest(composer)},

			expect: expect("1.10.0"),
		},
		{
			name: "Success/Proof",

			request: &core.PluginManifestGetRequest{Plugin: "scenes-composer", Version: "1.10.0"},

			daoRequest: &dao.PluginManifestSelectRequest{PluginVersionKey: composer},
			daoMock:    daoMock[*dao.PluginManifest]{resp: signedManifest},

			expect: expectSigned,
		},
		{
			name: "Success/Prerelease",

			request: &core.PluginManifestGetRequest{Plugin: "scenes-composer", Version: "2.0.0-dev.1"},

			daoRequest: &dao.PluginManifestSelectRequest{PluginVersionKey: composerDev},
			daoMock:    daoMock[*dao.PluginManifest]{resp: manifest(composerDev)},

			expect: expect("2.0.0-dev.1"),
		},
		{
			name: "Success/LongestPlugin",

			// A name holds 63 characters, digits and doubled hyphens included.
			request: &core.PluginManifestGetRequest{Plugin: "a2--" + strings.Repeat("b", 59), Version: "1.10.0"},

			daoRequest: &dao.PluginManifestSelectRequest{
				Plugin: "a2--" + strings.Repeat("b", 59), Major: 1, Minor: 10,
			},
			daoMock: daoMock[*dao.PluginManifest]{err: dao.ErrPluginVersionNotFound},

			expectErr: core.ErrPluginVersionNotFound,
		},
		{
			name: "Success/LargestNumber",

			// Each number fits a 32-bit column.
			request: &core.PluginManifestGetRequest{Plugin: "scenes-composer", Version: "2147483647.0.0"},

			daoRequest: &dao.PluginManifestSelectRequest{
				Plugin: "scenes-composer", Major: 2147483647,
			},
			daoMock: daoMock[*dao.PluginManifest]{err: dao.ErrPluginVersionNotFound},

			expectErr: core.ErrPluginVersionNotFound,
		},
		{
			name: "Error/MissingPlugin",

			request: &core.PluginManifestGetRequest{Version: "1.10.0"},

			expectErr: core.ErrInvalidRequest,
		},
		invalidPlugin("Uppercase", "Scenes-Composer"),
		invalidPlugin("Space", "scenes composer"),
		invalidPlugin("Underscore", "scenes_composer"),
		invalidPlugin("LeadingDigit", "3d-map"),
		invalidPlugin("LeadingHyphen", "-composer"),
		invalidPlugin("TrailingHyphen", "scenes-"),
		invalidPlugin("NotASCII", "scènes"),
		invalidPlugin("TooLong", strings.Repeat("a", 64)),
		invalid("Missing", ""),
		invalid("TwoNumbers", "1.10"),
		invalid("Prefix", "v1.10.0"),
		invalid("LeadingZero", "1.010.0"),
		invalid("NumberTooLarge", "2147483648.0.0"),
		invalid("BuildMetadata", "1.10.0+build.5"),
		invalid("EmptyPrerelease", "1.10.0-"),
		invalid("EmptyIdentifier", "1.10.0-dev..1"),
		invalid("NumericIdentifierLeadingZero", "1.10.0-dev.01"),
		invalid("Whitespace", "1.10.0 "),
		{
			name: "Error/Select",

			request: &core.PluginManifestGetRequest{Plugin: "scenes-composer", Version: "1.10.0"},

			daoRequest: &dao.PluginManifestSelectRequest{PluginVersionKey: composer},
			daoMock:    daoMock[*dao.PluginManifest]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockPluginManifestGetDao(t)

			if testCase.daoRequest != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, testCase.daoRequest).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			service := core.NewPluginManifestGet(mockDao)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)
		})
	}
}
