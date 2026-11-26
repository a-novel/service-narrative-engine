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

func TestPluginModuleGet(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	composer := dao.PluginVersionKey{Plugin: "scenes-composer", Major: 1, Minor: 10}

	testCases := []struct {
		name string

		request *core.PluginModuleGetRequest

		// daoRequest is the read the service asks for. The DAO is not called when it is nil.
		daoRequest *dao.PluginModuleSelectRequest
		daoMock    daoMock[*dao.PluginModule]

		expect    *core.PluginModule
		expectErr error
	}{
		{
			name: "Success/UI",

			request: &core.PluginModuleGetRequest{
				Plugin: "scenes-composer", Version: "1.10.0", Module: core.PluginModuleUI,
			},

			daoRequest: &dao.PluginModuleSelectRequest{PluginVersionKey: composer, Module: "ui"},
			daoMock: daoMock[*dao.PluginModule]{
				resp: &dao.PluginModule{Code: []byte("ui"), Checksum: "ui checksum", PublishedAt: createdAt},
			},

			expect: &core.PluginModule{Code: []byte("ui"), Checksum: "ui checksum", PublishedAt: createdAt},
		},
		{
			name: "Success/Migration",

			request: &core.PluginModuleGetRequest{
				Plugin: "scenes-composer", Version: "1.10.0", Module: core.PluginModuleMigration,
			},

			daoRequest: &dao.PluginModuleSelectRequest{PluginVersionKey: composer, Module: "migration"},
			daoMock: daoMock[*dao.PluginModule]{
				resp: &dao.PluginModule{Code: []byte("migration"), Checksum: "checksum", PublishedAt: createdAt},
			},

			expect: &core.PluginModule{Code: []byte("migration"), Checksum: "checksum", PublishedAt: createdAt},
		},
		{
			name: "Error/MissingPlugin",

			request: &core.PluginModuleGetRequest{Version: "1.10.0", Module: core.PluginModuleUI},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidPlugin",

			request: &core.PluginModuleGetRequest{Plugin: "Scenes Composer", Version: "1.10.0", Module: core.PluginModuleUI},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidVersion",

			request: &core.PluginModuleGetRequest{Plugin: "scenes-composer", Version: "1.10", Module: core.PluginModuleUI},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/MissingModule",

			request: &core.PluginModuleGetRequest{Plugin: "scenes-composer", Version: "1.10.0"},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/UnknownModule",

			request: &core.PluginModuleGetRequest{Plugin: "scenes-composer", Version: "1.10.0", Module: "schema"},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotFound",

			request: &core.PluginModuleGetRequest{
				Plugin: "scenes-composer", Version: "1.10.0", Module: core.PluginModuleMigration,
			},

			daoRequest: &dao.PluginModuleSelectRequest{PluginVersionKey: composer, Module: "migration"},
			daoMock:    daoMock[*dao.PluginModule]{err: dao.ErrPluginModuleNotFound},

			expectErr: core.ErrPluginModuleNotFound,
		},
		{
			name: "Error/Select",

			request: &core.PluginModuleGetRequest{
				Plugin: "scenes-composer", Version: "1.10.0", Module: core.PluginModuleUI,
			},

			daoRequest: &dao.PluginModuleSelectRequest{PluginVersionKey: composer, Module: "ui"},
			daoMock:    daoMock[*dao.PluginModule]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockPluginModuleGetDao(t)

			if testCase.daoRequest != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, testCase.daoRequest).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			service := core.NewPluginModuleGet(mockDao)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)
		})
	}
}
