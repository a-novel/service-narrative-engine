package core_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-narrative-engine/internal/core"
	coremocks "github.com/a-novel/service-narrative-engine/internal/core/mocks"
	"github.com/a-novel/service-narrative-engine/internal/dao"
)

func TestPluginMessagesGet(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	composer := dao.PluginVersionKey{Plugin: "scenes-composer", Major: 1, Minor: 10}

	testCases := []struct {
		name string

		request *core.PluginMessagesGetRequest

		// daoRequest is the read the service asks for. The DAO is not called when it is nil.
		daoRequest *dao.PluginMessagesSelectRequest
		daoMock    daoMock[*dao.PluginMessages]

		expect    *core.PluginMessages
		expectErr error
	}{
		{
			name: "Success",

			request: &core.PluginMessagesGetRequest{Plugin: "scenes-composer", Version: "1.10.0", Language: "fr"},

			daoRequest: &dao.PluginMessagesSelectRequest{PluginVersionKey: composer, Language: "fr"},
			daoMock: daoMock[*dao.PluginMessages]{resp: &dao.PluginMessages{
				PluginVersionKey: composer, Language: "fr", Title: "Compositeur de scènes", Description: "Compose.",
				Messages: []byte(`{"beats": "Temps"}`),
			}},

			expect: &core.PluginMessages{
				Language: "fr", Title: "Compositeur de scènes", Description: "Compose.",
				Messages: []byte(`{"beats": "Temps"}`),
			},
		},
		{
			name: "Error/MissingPlugin",

			request: &core.PluginMessagesGetRequest{Version: "1.10.0", Language: "fr"},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidPlugin",

			request: &core.PluginMessagesGetRequest{Plugin: "Scenes Composer", Version: "1.10.0", Language: "fr"},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidVersion",

			request: &core.PluginMessagesGetRequest{Plugin: "scenes-composer", Version: "1.10", Language: "fr"},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/MissingLanguage",

			request: &core.PluginMessagesGetRequest{Plugin: "scenes-composer", Version: "1.10.0"},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/InvalidLanguage",

			request: &core.PluginMessagesGetRequest{Plugin: "scenes-composer", Version: "1.10.0", Language: "french!"},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotFound",

			request: &core.PluginMessagesGetRequest{Plugin: "scenes-composer", Version: "1.10.0", Language: "de"},

			daoRequest: &dao.PluginMessagesSelectRequest{PluginVersionKey: composer, Language: "de"},
			daoMock:    daoMock[*dao.PluginMessages]{err: dao.ErrPluginMessagesNotFound},

			expectErr: core.ErrPluginMessagesNotFound,
		},
		{
			name: "Error/Select",

			request: &core.PluginMessagesGetRequest{Plugin: "scenes-composer", Version: "1.10.0", Language: "fr"},

			daoRequest: &dao.PluginMessagesSelectRequest{PluginVersionKey: composer, Language: "fr"},
			daoMock:    daoMock[*dao.PluginMessages]{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockPluginMessagesGetDao(t)

			if testCase.daoRequest != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, testCase.daoRequest).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			service := core.NewPluginMessagesGet(mockDao)

			result, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, result)
		})
	}
}
