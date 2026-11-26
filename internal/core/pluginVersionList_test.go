package core_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/v2/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/v2/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
)

func TestPluginVersionList(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	rows := []*dao.PluginVersionListRow{
		{
			Plugin: "scenes-composer", Major: 1, Minor: 2,
			HasMigration: true, SchemaChecksum: "first schema", Total: 2,
		},
		{
			Plugin: "scenes-composer", Major: 1, Minor: 10,
			SchemaChecksum: "second schema", Total: 2,
		},
	}

	page := &core.PluginVersionPage{
		Versions: []*core.PluginVersion{
			{Version: "1.2.0", HasMigration: true, SchemaChecksum: "first schema"},
			{Version: "1.10.0", SchemaChecksum: "second schema"},
		},
		Total: 2,
	}

	testCases := []struct {
		name string

		request *core.PluginVersionListRequest

		// daoRequest is the read the service asks for. The DAO is not called when it is nil.
		daoRequest *dao.PluginVersionListRequest
		daoMock    daoMock[[]*dao.PluginVersionListRow]

		expect    *core.PluginVersionPage
		expectErr error
	}{
		{
			name: "Success",

			request: &core.PluginVersionListRequest{Plugin: "scenes-composer", After: "1.0.0", Limit: 10, Offset: 5},

			daoRequest: &dao.PluginVersionListRequest{
				After: dao.PluginVersionKey{Plugin: "scenes-composer", Major: 1}, Limit: 10, Offset: 5,
			},
			daoMock: daoMock[[]*dao.PluginVersionListRow]{resp: rows},

			expect: page,
		},
		{
			name: "Success/AfterPrerelease",

			// The prerelease reaches the DAO, which lists the release of 1.2.0 after it. No limit takes the
			// default page size.
			request: &core.PluginVersionListRequest{Plugin: "scenes-composer", After: "1.2.0-dev.1"},

			daoRequest: &dao.PluginVersionListRequest{
				After: dao.PluginVersionKey{Plugin: "scenes-composer", Major: 1, Minor: 2, Prerelease: "dev.1"},
				Limit: core.PluginVersionListLimitDefault,
			},
			daoMock: daoMock[[]*dao.PluginVersionListRow]{resp: rows},

			expect: page,
		},
		{
			name: "Success/Empty",

			request: &core.PluginVersionListRequest{Plugin: "scenes-composer", After: "1.10.0", Limit: 500},

			daoRequest: &dao.PluginVersionListRequest{
				After: dao.PluginVersionKey{Plugin: "scenes-composer", Major: 1, Minor: 10}, Limit: 500,
			},

			expect: &core.PluginVersionPage{Versions: []*core.PluginVersion{}},
		},
		{
			name: "Error/MissingPlugin",

			request: &core.PluginVersionListRequest{After: "1.0.0"},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidPlugin",

			request: &core.PluginVersionListRequest{Plugin: "Scenes Composer", After: "1.0.0"},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/MissingAfter",

			request: &core.PluginVersionListRequest{Plugin: "scenes-composer"},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidAfter",

			request: &core.PluginVersionListRequest{Plugin: "scenes-composer", After: "1.0"},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NegativeLimit",

			request: &core.PluginVersionListRequest{Plugin: "scenes-composer", After: "1.0.0", Limit: -1},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/LimitTooLarge",

			request: &core.PluginVersionListRequest{Plugin: "scenes-composer", After: "1.0.0", Limit: 501},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NegativeOffset",

			request: &core.PluginVersionListRequest{Plugin: "scenes-composer", After: "1.0.0", Offset: -1},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/List",

			request: &core.PluginVersionListRequest{Plugin: "scenes-composer", After: "1.0.0", Limit: 10},

			daoRequest: &dao.PluginVersionListRequest{
				After: dao.PluginVersionKey{Plugin: "scenes-composer", Major: 1}, Limit: 10,
			},
			daoMock: daoMock[[]*dao.PluginVersionListRow]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockPluginVersionListDao(t)

			if testCase.daoRequest != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, testCase.daoRequest).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			service := core.NewPluginVersionList(mockDao)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)
		})
	}
}
