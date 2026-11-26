package dao_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/v2/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
	"github.com/a-novel/service-narrative-engine/v2/internal/models/migrations"
)

func TestPgPluginModuleSelect(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *dao.PluginModuleSelectRequest

		expect    *dao.PluginModule
		expectErr error
	}{
		{
			name: "Success/UI",

			request: &dao.PluginModuleSelectRequest{PluginVersionKey: composer120, Module: "ui"},

			expect: &dao.PluginModule{
				Code: pluginUI(composer120), Checksum: pluginChecksum(pluginUI(composer120)), PublishedAt: createdAt,
			},
		},
		{
			name: "Success/Migration",

			request: &dao.PluginModuleSelectRequest{PluginVersionKey: composer120, Module: "migration"},

			expect: &dao.PluginModule{
				Code: pluginMigration(composer120), Checksum: pluginChecksum(pluginMigration(composer120)),
				PublishedAt: createdAt,
			},
		},
		{
			name: "Error/NoMigration",

			request: &dao.PluginModuleSelectRequest{PluginVersionKey: composer100, Module: "migration"},

			expectErr: dao.ErrPluginModuleNotFound,
		},
		{
			name: "Error/UnknownModule",

			request: &dao.PluginModuleSelectRequest{PluginVersionKey: composer120, Module: "schema"},

			expectErr: dao.ErrPluginModuleNotFound,
		},
		{
			name: "Error/AbsentVersion",

			request: &dao.PluginModuleSelectRequest{
				Plugin: "scenes-composer", Major: 1, Minor: 1, Module: "ui",
			},

			expectErr: dao.ErrPluginModuleNotFound,
		},
	}

	daoPluginModuleSelect := dao.NewPgPluginModuleSelect()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, pluginFixtures())

				result, err := daoPluginModuleSelect.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
