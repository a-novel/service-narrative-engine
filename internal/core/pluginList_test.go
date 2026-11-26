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

func TestPluginList(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	rows := []*dao.PluginListRow{
		{
			Plugin: "characters", Minor: 1,
			Title: "Characters", Description: "Who appears.", Total: 2,
		},
		{
			Plugin: "scenes-composer", Major: 1, Minor: 10,
			Title: "Compositeur de scènes", Description: "Compose.", Total: 2,
		},
	}

	page := &core.PluginPage{
		Plugins: []*core.Plugin{
			{Name: "characters", Version: "0.1.0", Title: "Characters", Description: "Who appears."},
			{Name: "scenes-composer", Version: "1.10.0", Title: "Compositeur de scènes", Description: "Compose."},
		},
		Total: 2,
	}

	testCases := []struct {
		name string

		request *core.PluginListRequest

		// daoRequest is the read the service asks for. The DAO is not called when it is nil.
		daoRequest *dao.PluginListRequest
		daoMock    daoMock[[]*dao.PluginListRow]

		expect    *core.PluginPage
		expectErr error
	}{
		{
			name: "Success",

			request: &core.PluginListRequest{Language: "fr", Limit: 10, Offset: 5},

			daoRequest: &dao.PluginListRequest{Language: "fr", Limit: 10, Offset: 5},
			daoMock:    daoMock[[]*dao.PluginListRow]{resp: rows},

			expect: page,
		},
		{
			name: "Success/Defaults",

			// No language reads in English, and no limit takes the default page size.
			request: &core.PluginListRequest{},

			daoRequest: &dao.PluginListRequest{Limit: core.PluginListLimitDefault},
			daoMock:    daoMock[[]*dao.PluginListRow]{resp: rows},

			expect: page,
		},
		{
			name: "Success/Empty",

			request: &core.PluginListRequest{Limit: 500, Offset: 2},

			daoRequest: &dao.PluginListRequest{Limit: 500, Offset: 2},

			expect: &core.PluginPage{Plugins: []*core.Plugin{}},
		},
		{
			name: "Error/InvalidLanguage",

			request: &core.PluginListRequest{Language: "french!"},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NegativeLimit",

			request: &core.PluginListRequest{Limit: -1},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/LimitTooLarge",

			request: &core.PluginListRequest{Limit: 501},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NegativeOffset",

			request: &core.PluginListRequest{Offset: -1},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/List",

			request: &core.PluginListRequest{Limit: 10},

			daoRequest: &dao.PluginListRequest{Limit: 10},
			daoMock:    daoMock[[]*dao.PluginListRow]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockPluginListDao(t)

			if testCase.daoRequest != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, testCase.daoRequest).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			service := core.NewPluginList(mockDao)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)
		})
	}
}
