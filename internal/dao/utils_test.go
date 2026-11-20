package dao_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

var (
	ownerID      = uuid.MustParse("00000000-0000-0000-0000-0000000000a1")
	otherOwnerID = uuid.MustParse("00000000-0000-0000-0000-0000000000a2")

	projectID      = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	otherProjectID = uuid.MustParse("00000000-0000-0000-0000-000000000002")

	// absentID names no row of the fixtures.
	absentID = uuid.MustParse("00000000-0000-0000-0000-0000000000ff")

	createdAt = time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt = time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)
	now       = time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC)
)

// The versions the plugin fixtures publish. The composer's releases prove that versions order by number: as
// text, 1.10.0 would come before 1.2.0.
var (
	composer100    = dao.PluginVersionKey{Plugin: "scenes-composer", Major: 1}
	composer120    = dao.PluginVersionKey{Plugin: "scenes-composer", Major: 1, Minor: 2}
	composer1100   = dao.PluginVersionKey{Plugin: "scenes-composer", Major: 1, Minor: 10}
	composer200Dev = dao.PluginVersionKey{Plugin: "scenes-composer", Major: 2, Prerelease: "dev.1"}
	characters010  = dao.PluginVersionKey{Plugin: "characters", Minor: 1}
	drafts010Dev   = dao.PluginVersionKey{Plugin: "drafts", Minor: 1, Prerelease: "dev.1"}
	glossary010    = dao.PluginVersionKey{Plugin: "glossary", Minor: 1}
)

// The JSON documents of the plugin fixtures, spelled as PostgreSQL returns a jsonb value, so a read compares
// equal to its fixture.
const (
	pluginSchema     = `{"type": "object"}`
	pluginSchemaNext = `{"type": "object", "required": ["beats"]}`
	pluginExports    = `{"scenes": {"id": "/id", "select": "$.beats[*].scenes[*]"}}`
)

// fixtures is the data a DAO test seeds before it runs the operation under test.
type fixtures struct {
	projects []*dao.Project
	logos    []*dao.ProjectLogo

	pluginVersions []*dao.PluginVersion
	pluginMessages []*dao.PluginMessages
}

// projectFixtures returns a project beside a project of another owner. Each call returns fresh values, so a
// case may change them.
func projectFixtures() fixtures {
	return fixtures{
		projects: []*dao.Project{
			{
				ID: projectID, OwnerID: ownerID, Title: "Les Heures claires", Premise: "Marie attend.",
				Language: "fr", Tags: []string{"noir", "huis clos"}, Genres: []string{"thriller"},
				CreatedAt: createdAt, UpdatedAt: updatedAt, EditedAt: updatedAt,
			},
			{
				ID: otherProjectID, OwnerID: otherOwnerID, Title: "Other", Language: "en", Tags: []string{},
				Genres: []string{}, CreatedAt: createdAt, UpdatedAt: updatedAt, EditedAt: updatedAt,
			},
		},
	}
}

// logoFixtures returns the project fixtures with a logo on the test project. The logo and the project carry
// the logo's checksum, as a read returns them.
func logoFixtures() fixtures {
	data := projectFixtures()
	data.logos = []*dao.ProjectLogo{
		{
			ProjectID: projectID, Data: []byte("stored logo"), MediaType: "image/webp",
			Checksum: *logoChecksum([]byte("stored logo")), UpdatedAt: updatedAt,
		},
	}
	data.projects[0].LogoChecksum = logoChecksum(data.logos[0].Data)

	return data
}

// logoChecksum returns the checksum a read joins for a logo of these bytes: their SHA-256, in lowercase hex.
func logoChecksum(data []byte) *string {
	sum := sha256.Sum256(data)

	return new(hex.EncodeToString(sum[:]))
}

// pluginFixtures returns a registry of four plugins. The composer has three releases and a later prerelease,
// and its latest release is translated in French. Characters has one release, in English alone. Drafts only
// has a prerelease. Glossary has one release, in Spanish and Italian but not in English. Each call returns
// fresh values, so a case may change them.
func pluginFixtures() fixtures {
	version := func(key dao.PluginVersionKey, schema string, migration []byte) *dao.PluginVersion {
		return &dao.PluginVersion{
			PluginVersionKey: key, Schema: json.RawMessage(schema), Exports: json.RawMessage(pluginExports),
			Migration: migration, UI: pluginUI(key), PublishedAt: createdAt,
		}
	}

	return fixtures{
		pluginVersions: []*dao.PluginVersion{
			version(composer100, pluginSchema, nil),
			version(composer120, pluginSchema, pluginMigration(composer120)),
			version(composer1100, pluginSchemaNext, pluginMigration(composer1100)),
			version(composer200Dev, pluginSchemaNext, nil),
			version(characters010, pluginSchema, nil),
			version(drafts010Dev, pluginSchema, nil),
			version(glossary010, pluginSchema, nil),
		},
		pluginMessages: []*dao.PluginMessages{
			pluginMessages(composer100, "en", "Scenes Composer"),
			pluginMessages(composer120, "en", "Scenes Composer"),
			pluginMessages(composer1100, "en", "Scenes Composer"),
			pluginMessages(composer1100, "fr", "Compositeur de scènes"),
			pluginMessages(composer200Dev, "en", "Scenes Composer"),
			pluginMessages(characters010, "en", "Characters"),
			pluginMessages(drafts010Dev, "en", "Drafts"),
			pluginMessages(glossary010, "es", "Glosario"),
			pluginMessages(glossary010, "it", "Glossario"),
		},
	}
}

// pluginUI returns the UI module the fixtures give a version, which names it.
func pluginUI(key dao.PluginVersionKey) []byte {
	return fmt.Appendf(nil, "export default %q;", fmt.Sprint("ui ", key))
}

// pluginMigration returns the migration module the fixtures give a version that ships one.
func pluginMigration(key dao.PluginVersionKey) []byte {
	return fmt.Appendf(nil, "export default %q;", fmt.Sprint("migration ", key))
}

// pluginMessages returns the translations the fixtures give a version in a language.
func pluginMessages(key dao.PluginVersionKey, language, title string) *dao.PluginMessages {
	return &dao.PluginMessages{
		PluginVersionKey: key, Language: language, Title: title, Description: "About " + title,
		Messages: json.RawMessage(`{"title": "` + title + `"}`),
	}
}

// pluginChecksum returns the checksum a read returns for a module of these bytes: their SHA-256, in lowercase
// hex.
func pluginChecksum(data []byte) string {
	sum := sha256.Sum256(data)

	return hex.EncodeToString(sum[:])
}

// pluginSchemaChecksum returns the checksum a read returns for a schema. The registry hashes what jsonb_send
// writes: a version byte, then the schema as PostgreSQL spells it.
func pluginSchemaChecksum(schema string) string {
	return pluginChecksum(append([]byte{1}, schema...))
}

// insertFixtures seeds data in the order its foreign keys need. Parallel tests insert the same fixture values, so
// each insert returns nothing: scanning the returned rows into the shared values would race with another test
// reading them.
func insertFixtures(ctx context.Context, t *testing.T, data fixtures) {
	t.Helper()

	db, err := postgres.GetContext(ctx)
	require.NoError(t, err)

	if len(data.projects) > 0 {
		_, err = db.NewInsert().Model(&data.projects).Returning("NULL").Exec(ctx)
		require.NoError(t, err)
	}

	if len(data.logos) > 0 {
		_, err = db.NewInsert().Model(&data.logos).Returning("NULL").Exec(ctx)
		require.NoError(t, err)
	}

	if len(data.pluginVersions) > 0 {
		_, err = db.NewInsert().Model(&data.pluginVersions).Returning("NULL").Exec(ctx)
		require.NoError(t, err)
	}

	if len(data.pluginMessages) > 0 {
		_, err = db.NewInsert().Model(&data.pluginMessages).Returning("NULL").Exec(ctx)
		require.NoError(t, err)
	}
}

// projectLogo reads the logo of the test project, nil when it holds none.
func projectLogo(ctx context.Context, t *testing.T) *dao.ProjectLogo {
	t.Helper()

	logo, err := dao.NewPgProjectLogoSelect().Exec(ctx, &dao.ProjectLogoSelectRequest{ProjectID: projectID})
	if errors.Is(err, dao.ErrProjectLogoSelectNotFound) {
		return nil
	}

	require.NoError(t, err)

	return logo
}
