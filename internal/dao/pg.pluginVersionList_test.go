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

func TestPgPluginVersionList(t *testing.T) {
	t.Parallel()

	// The composer has two releases after its first one.
	releases := []*dao.PluginVersionListRow{
		{
			PluginVersionKey: composer120, HasMigration: true, SchemaChecksum: pluginSchemaChecksum(pluginSchema), Total: 2,
		},
		{
			PluginVersionKey: composer1100, HasMigration: true, SchemaChecksum: pluginSchemaChecksum(pluginSchemaNext),
			Total: 2,
		},
	}

	testCases := []struct {
		name string

		request *dao.PluginVersionListRequest

		expect    []*dao.PluginVersionListRow
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.PluginVersionListRequest{After: composer100, Limit: 10},

			// 1.10.0 comes after 1.2.0, and the prerelease of 2.0.0 is left out.
			expect: releases,
		},
		{
			name: "Success/AfterAbsentVersion",

			request: &dao.PluginVersionListRequest{
				After: dao.PluginVersionKey{Plugin: "scenes-composer", Major: 1, Minor: 1}, Limit: 10,
			},

			expect: releases,
		},
		{
			name: "Success/AfterPrerelease",

			// The release of 1.2.0 comes after its prereleases.
			request: &dao.PluginVersionListRequest{
				After: dao.PluginVersionKey{Plugin: "scenes-composer", Major: 1, Minor: 2, Prerelease: "dev.1"}, Limit: 10,
			},

			expect: releases,
		},
		{
			name: "Success/AfterRelease",

			request: &dao.PluginVersionListRequest{After: composer120, Limit: 10},

			expect: []*dao.PluginVersionListRow{
				{
					PluginVersionKey: composer1100, HasMigration: true,
					SchemaChecksum: pluginSchemaChecksum(pluginSchemaNext), Total: 1,
				},
			},
		},
		{
			name: "Success/AfterLatest",

			request: &dao.PluginVersionListRequest{After: composer1100, Limit: 10},
		},
		{
			name: "Success/Page",

			request: &dao.PluginVersionListRequest{After: composer100, Limit: 1, Offset: 1},

			expect: releases[1:],
		},
		{
			name: "Success/PastEnd",

			request: &dao.PluginVersionListRequest{After: composer100, Limit: 10, Offset: 2},
		},
		{
			name: "Success/AbsentPlugin",

			request: &dao.PluginVersionListRequest{After: dao.PluginVersionKey{Plugin: "absent"}, Limit: 10},
		},
	}

	daoPluginVersionList := dao.NewPgPluginVersionList()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, pluginFixtures())

				result, err := daoPluginVersionList.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
