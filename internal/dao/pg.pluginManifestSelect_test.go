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

func TestPgPluginManifestSelect(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *dao.PluginManifestSelectRequest

		expect    *dao.PluginManifest
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.PluginManifestSelectRequest{PluginVersionKey: composer1100},

			expect: &dao.PluginManifest{
				PluginVersionKey: composer1100, Schema: []byte(pluginSchemaNext), Exports: []byte(pluginExports),
				HasMigration: true, SchemaChecksum: pluginSchemaChecksum(pluginSchemaNext),
				UIChecksum: pluginChecksum(pluginUI(composer1100)), PublishedAt: createdAt, PublishedBy: "fixture",
			},
		},
		{
			name: "Success/NoMigration",

			request: &dao.PluginManifestSelectRequest{PluginVersionKey: composer100},

			expect: &dao.PluginManifest{
				PluginVersionKey: composer100, Schema: []byte(pluginSchema), Exports: []byte(pluginExports),
				SchemaChecksum: pluginSchemaChecksum(pluginSchema), UIChecksum: pluginChecksum(pluginUI(composer100)),
				PublishedAt: createdAt, PublishedBy: "fixture",
			},
		},
		{
			name: "Success/Prerelease",

			request: &dao.PluginManifestSelectRequest{PluginVersionKey: composer200Dev},

			expect: &dao.PluginManifest{
				PluginVersionKey: composer200Dev, Schema: []byte(pluginSchemaNext), Exports: []byte(pluginExports),
				SchemaChecksum: pluginSchemaChecksum(pluginSchemaNext),
				UIChecksum:     pluginChecksum(pluginUI(composer200Dev)), PublishedAt: createdAt, PublishedBy: "fixture",
			},
		},
		{
			name: "Error/AbsentVersion",

			request: &dao.PluginManifestSelectRequest{
				Plugin: "scenes-composer", Major: 1, Minor: 1,
			},

			expectErr: dao.ErrPluginVersionNotFound,
		},
		{
			name: "Error/AbsentPrerelease",

			// The release exists, and no prerelease of it does.
			request: &dao.PluginManifestSelectRequest{
				Plugin: "scenes-composer", Major: 1, Prerelease: "dev.1",
			},

			expectErr: dao.ErrPluginVersionNotFound,
		},
		{
			name: "Error/AbsentPlugin",

			request: &dao.PluginManifestSelectRequest{Plugin: "absent", Major: 1},

			expectErr: dao.ErrPluginVersionNotFound,
		},
	}

	daoPluginManifestSelect := dao.NewPgPluginManifestSelect()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, pluginFixtures())

				result, err := daoPluginManifestSelect.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
