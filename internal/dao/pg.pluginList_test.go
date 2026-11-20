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

func TestPgPluginList(t *testing.T) {
	t.Parallel()

	// Drafts only has a prerelease, so the registry lists three plugins.
	row := func(key dao.PluginVersionKey, title string) *dao.PluginListRow {
		return &dao.PluginListRow{PluginVersionKey: key, Title: title, Description: "About " + title, Total: 3}
	}

	testCases := []struct {
		name string

		request *dao.PluginListRequest

		expect    []*dao.PluginListRow
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.PluginListRequest{Language: "en", Limit: 10},

			// The composer's latest release is 1.10.0: above 1.2.0, and below the prerelease of 2.0.0. Glossary
			// is not translated in English, so it reads in the first of its languages, Spanish.
			expect: []*dao.PluginListRow{
				row(characters010, "Characters"), row(glossary010, "Glosario"), row(composer1100, "Scenes Composer"),
			},
		},
		{
			name: "Success/Language",

			request: &dao.PluginListRequest{Language: "fr", Limit: 10},

			// Characters is not translated in French, so it reads in English.
			expect: []*dao.PluginListRow{
				row(characters010, "Characters"),
				row(glossary010, "Glosario"),
				row(composer1100, "Compositeur de scènes"),
			},
		},
		{
			name: "Success/LanguageWithoutEnglish",

			request: &dao.PluginListRequest{Language: "it", Limit: 10},

			// The asked language comes before the first one.
			expect: []*dao.PluginListRow{
				row(characters010, "Characters"), row(glossary010, "Glossario"), row(composer1100, "Scenes Composer"),
			},
		},
		{
			name: "Success/UnknownLanguage",

			request: &dao.PluginListRequest{Language: "de", Limit: 10},

			expect: []*dao.PluginListRow{
				row(characters010, "Characters"), row(glossary010, "Glosario"), row(composer1100, "Scenes Composer"),
			},
		},
		{
			name: "Success/NoLanguage",

			request: &dao.PluginListRequest{Limit: 10},

			expect: []*dao.PluginListRow{
				row(characters010, "Characters"), row(glossary010, "Glosario"), row(composer1100, "Scenes Composer"),
			},
		},
		{
			name: "Success/Page",

			request: &dao.PluginListRequest{Language: "en", Limit: 1, Offset: 1},

			expect: []*dao.PluginListRow{row(glossary010, "Glosario")},
		},
		{
			name: "Success/PastEnd",

			request: &dao.PluginListRequest{Language: "en", Limit: 10, Offset: 3},
		},
	}

	daoPluginList := dao.NewPgPluginList()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, pluginFixtures())

				result, err := daoPluginList.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
