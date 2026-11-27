package dao_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/v2/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
	"github.com/a-novel/service-narrative-engine/v2/internal/models/migrations"
)

func TestPgPluginVersionInsert(t *testing.T) {
	t.Parallel()

	composer1110 := dao.PluginVersionKey{Plugin: "scenes-composer", Major: 1, Minor: 11}
	composer100Dev := dao.PluginVersionKey{Plugin: "scenes-composer", Major: 1, Prerelease: "dev.1"}
	timeline100 := dao.PluginVersionKey{Plugin: "timeline", Major: 1}

	english := dao.PluginVersionInsertLanguage{
		Language: "en", Title: "Scenes Composer", Description: "About Scenes Composer",
		Messages: json.RawMessage(`{"title": "Scenes Composer"}`),
	}
	french := dao.PluginVersionInsertLanguage{
		Language: "fr", Title: "Compositeur de scènes", Description: "About Compositeur de scènes",
		Messages: json.RawMessage(`{"title": "Compositeur de scènes"}`),
	}

	request := func(
		key dao.PluginVersionKey, migration []byte, languages ...dao.PluginVersionInsertLanguage,
	) *dao.PluginVersionInsertRequest {
		return &dao.PluginVersionInsertRequest{
			PluginVersionKey: key, Schema: json.RawMessage(pluginSchemaNext), Exports: json.RawMessage(pluginExports),
			Migration: migration, UI: pluginUI(key), Languages: languages, Now: now,
		}
	}

	// stored returns the rows a request stores.
	stored := func(request *dao.PluginVersionInsertRequest) (*dao.PluginVersion, []*dao.PluginMessages) {
		version := &dao.PluginVersion{
			PluginVersionKey: request.PluginVersionKey, Schema: request.Schema, Exports: request.Exports,
			Migration: request.Migration, UI: request.UI, Proof: request.Proof, APIKeyID: request.APIKeyID,
			PublishedAt: request.Now,
		}

		messages := make([]*dao.PluginMessages, len(request.Languages))
		for i, language := range request.Languages {
			messages[i] = &dao.PluginMessages{
				PluginVersionKey: request.PluginVersionKey, Language: language.Language, Title: language.Title,
				Description: language.Description, Messages: language.Messages,
			}
		}

		return version, messages
	}

	signed := request(composer1110, pluginMigration(composer1110), english)
	signed.Proof, signed.APIKeyID = new(pluginProof), new(pluginAPIKeyID)

	untitled := english
	untitled.Title = ""

	testCases := []struct {
		name string

		request *dao.PluginVersionInsertRequest

		expectErr error
	}{
		{
			name: "Success",

			request: request(composer1110, pluginMigration(composer1110), english, french),
		},
		{
			// A version published with an API key stores its proof and the ID of the key.
			name: "Success/Proof",

			request: signed,
		},
		{
			name: "Success/NoMigration",

			request: request(composer1110, nil, english),
		},
		{
			name: "Success/NewPlugin",

			request: request(timeline100, nil, english),
		},
		{
			// A prerelease is a version of its own beside its release.
			name: "Success/Prerelease",

			request: request(composer100Dev, nil, english),
		},
		{
			name: "Error/Exists",

			// The registry keeps the version it holds, whatever the new one carries.
			request: request(composer100, pluginMigration(composer100), english, french),

			expectErr: dao.ErrPluginVersionExists,
		},
		{
			name: "Error/ExistsPrerelease",

			request: request(composer200Dev, nil, english),

			expectErr: dao.ErrPluginVersionExists,
		},
	}

	daoPluginVersionInsert := dao.NewPgPluginVersionInsert()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				data := pluginFixtures()
				insertFixtures(ctx, t, data)

				err := daoPluginVersionInsert.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)

				expectVersions, expectMessages := data.pluginVersions, data.pluginMessages

				if testCase.expectErr == nil {
					version, messages := stored(testCase.request)
					expectVersions, expectMessages = append(expectVersions, version), append(expectMessages, messages...)
				}

				versions, messages := listRegistry(ctx, t)
				require.ElementsMatch(t, expectVersions, versions)
				require.ElementsMatch(t, expectMessages, messages)
			})
		})
	}

	// A translation the schema refuses stores neither itself, nor the others, nor the version.
	t.Run("Error/StoresNothing", func(t *testing.T) {
		t.Parallel()

		postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
			t.Helper()

			err := daoPluginVersionInsert.Exec(ctx, request(composer1110, nil, english, untitled))
			require.Error(t, err)
			require.NotErrorIs(t, err, dao.ErrPluginVersionExists)

			versions, messages := listRegistry(ctx, t)
			require.Empty(t, versions)
			require.Empty(t, messages)
		})
	})
}

// listRegistry reads every version of the registry and every translation, in no particular order.
func listRegistry(ctx context.Context, t *testing.T) ([]*dao.PluginVersion, []*dao.PluginMessages) {
	t.Helper()

	db, err := postgres.GetContext(ctx)
	require.NoError(t, err)

	versions := []*dao.PluginVersion{}
	require.NoError(t, db.NewSelect().Model(&versions).Scan(ctx))

	messages := []*dao.PluginMessages{}
	require.NoError(t, db.NewSelect().Model(&messages).Scan(ctx))

	return versions, messages
}
