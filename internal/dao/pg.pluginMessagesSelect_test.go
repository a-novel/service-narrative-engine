package dao_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgPluginMessagesSelect(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *dao.PluginMessagesSelectRequest

		expect    *dao.PluginMessages
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.PluginMessagesSelectRequest{PluginVersionKey: composer1100, Language: "fr"},

			expect: pluginMessages(composer1100, "fr", "Compositeur de scènes"),
		},
		{
			name: "Success/English",

			request: &dao.PluginMessagesSelectRequest{PluginVersionKey: composer1100, Language: "en"},

			expect: pluginMessages(composer1100, "en", "Scenes Composer"),
		},
		{
			name: "Error/NoLanguage",

			// The version exists, and is not translated in French.
			request: &dao.PluginMessagesSelectRequest{PluginVersionKey: characters010, Language: "fr"},

			expectErr: dao.ErrPluginMessagesNotFound,
		},
		{
			name: "Error/AbsentVersion",

			request: &dao.PluginMessagesSelectRequest{
				Plugin: "scenes-composer", Major: 1, Minor: 1, Language: "en",
			},

			expectErr: dao.ErrPluginMessagesNotFound,
		},
	}

	daoPluginMessagesSelect := dao.NewPgPluginMessagesSelect()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, pluginFixtures())

				result, err := daoPluginMessagesSelect.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
