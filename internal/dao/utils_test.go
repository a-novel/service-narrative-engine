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

	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
)

var (
	ownerID      = uuid.MustParse("00000000-0000-0000-0000-0000000000a1")
	otherOwnerID = uuid.MustParse("00000000-0000-0000-0000-0000000000a2")

	projectID      = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	otherProjectID = uuid.MustParse("00000000-0000-0000-0000-000000000002")

	// The instances of the fixtures, in the order their IDs list them. The test project holds the first two and
	// opens on the first. The project of the other owner holds the third.
	instanceID      = uuid.MustParse("00000000-0000-0000-0000-000000000031")
	otherInstanceID = uuid.MustParse("00000000-0000-0000-0000-000000000032")
	alienInstanceID = uuid.MustParse("00000000-0000-0000-0000-000000000041")

	// absentID names no row of the fixtures.
	absentID = uuid.MustParse("00000000-0000-0000-0000-0000000000ff")

	createdAt = time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt = time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)
	now       = time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC)

	// documentUpdatedAt is when the documents of the fixtures last changed, to the microsecond PostgreSQL
	// keeps. A write names it to apply.
	documentUpdatedAt = time.Date(2021, 1, 2, 0, 0, 0, 123456000, time.UTC)
)

// storedBytesQuery reads, for each project, the count of the bytes its rows hold, and a fresh sum of them.
const storedBytesQuery = `
SELECT projects.id AS project_id, projects.stored_bytes AS stored, coalesce(sum(held.bytes), 0)::bigint AS bytes
FROM projects
LEFT JOIN (
  SELECT project_id, bytes FROM instances
  UNION ALL SELECT project_id, bytes FROM project_links
) AS held ON held.project_id = projects.id
GROUP BY projects.id`

// storedBytes is the count of the bytes a project's rows hold, beside a fresh sum of them.
type storedBytes struct {
	ProjectID uuid.UUID `bun:"project_id,type:uuid"`
	// Stored is the count the schema's triggers keep.
	Stored int64 `bun:"stored"`
	// Bytes is the fresh sum.
	Bytes int64 `bun:"bytes"`
}

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

	instanceContent     = `{"beats": []}`
	instanceContentNext = `{"beats": [{"id": "a"}]}`
	projectLinks        = `[{"source": {"id": "00000000-0000-0000-0000-000000000031", "target": "scenes/a"}, ` +
		`"destination": {"id": "00000000-0000-0000-0000-000000000032", "target": "characters/b"}, ` +
		`"relationship": "therefore"}]`
)

// The proof the plugin fixtures give a version published with an API key, and the ID of that key.
const (
	pluginProof    = "header.claims.signature"
	pluginAPIKeyID = "publisher-key"
)

// fixtures is the data a DAO test seeds before it runs the operation under test.
type fixtures struct {
	projects []*dao.Project
	logos    []*dao.ProjectLogo

	instances []*dao.Instance
	links     []*dao.ProjectLinks

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

// instanceFixtures returns the project fixtures with instances and links. The test project holds two instances,
// opens on the first and holds links. The project of the other owner opens on an instance of its own, and never
// wrote its links.
func instanceFixtures() fixtures {
	data := projectFixtures()
	data.instances = []*dao.Instance{
		{
			ID: instanceID, ProjectID: projectID, Plugin: "scenes-composer", Version: "1.0.0", Name: "Composer",
			CreatedAt: createdAt, UpdatedAt: documentUpdatedAt,
			Content: json.RawMessage(instanceContent),
		},
		{
			ID: otherInstanceID, ProjectID: projectID, Plugin: "characters", Version: "0.1.0-dev.1",
			Name: "Characters", CreatedAt: createdAt, UpdatedAt: documentUpdatedAt,
			Content: json.RawMessage(instanceContentNext),
		},
		{
			ID: alienInstanceID, ProjectID: otherProjectID, Plugin: "scenes-composer", Version: "1.0.0",
			Name: "Composer", CreatedAt: createdAt, UpdatedAt: documentUpdatedAt,
			Content: json.RawMessage(instanceContent),
		},
	}
	data.links = []*dao.ProjectLinks{
		{ProjectID: projectID, Links: json.RawMessage(projectLinks), UpdatedAt: documentUpdatedAt},
	}
	data.projects[0].DefaultInstanceID = new(instanceID)
	data.projects[1].DefaultInstanceID = new(alienInstanceID)

	return data
}

// descriptors returns the fixture instances with these IDs, without their documents.
func (data fixtures) descriptors(ids ...uuid.UUID) []*dao.InstanceDescriptor {
	found := make([]*dao.InstanceDescriptor, 0, len(ids))

	for _, id := range ids {
		for _, instance := range data.instances {
			if instance.ID == id {
				found = append(found, &instance.InstanceDescriptor)
			}
		}
	}

	return found
}

// page returns the fixture instances with these IDs as the rows of a page, each with the total of its list.
func (data fixtures) page(total int, ids ...uuid.UUID) []*dao.InstanceListRow {
	rows := make([]*dao.InstanceListRow, 0, len(ids))

	for _, descriptor := range data.descriptors(ids...) {
		rows = append(rows, &dao.InstanceListRow{InstanceDescriptor: *descriptor, Total: total})
	}

	return rows
}

// logoChecksum returns the checksum a read joins for a logo of these bytes: their SHA-256, in lowercase hex.
func logoChecksum(data []byte) *string {
	sum := sha256.Sum256(data)

	return new(hex.EncodeToString(sum[:]))
}

// pluginFixtures returns a registry of four plugins. The composer has three releases and a later prerelease.
// Its latest release is translated in French, and is the one version published with an API key. Characters
// has one release, in English alone. Drafts only has a prerelease. Glossary has one release, in Spanish and
// Italian but not in English. Each call returns fresh values, so a case may change them.
func pluginFixtures() fixtures {
	version := func(key dao.PluginVersionKey, schema string, migration []byte) *dao.PluginVersion {
		return &dao.PluginVersion{
			PluginVersionKey: key, Schema: json.RawMessage(schema), Exports: json.RawMessage(pluginExports),
			Migration: migration, UI: pluginUI(key), PublishedAt: createdAt,
		}
	}

	signed := version(composer1100, pluginSchemaNext, pluginMigration(composer1100))
	signed.Proof, signed.APIKeyID = new(pluginProof), new(pluginAPIKeyID)

	return fixtures{
		pluginVersions: []*dao.PluginVersion{
			version(composer100, pluginSchema, nil),
			version(composer120, pluginSchema, pluginMigration(composer120)),
			signed,
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

	// A project names its default instance once its instances exist.
	if len(data.projects) > 0 {
		_, err = db.NewInsert().Model(&data.projects).ExcludeColumn("default_instance_id").Returning("NULL").Exec(ctx)
		require.NoError(t, err)
	}

	if len(data.logos) > 0 {
		_, err = db.NewInsert().Model(&data.logos).Returning("NULL").Exec(ctx)
		require.NoError(t, err)
	}

	if len(data.instances) > 0 {
		_, err = db.NewInsert().Model(&data.instances).Returning("NULL").Exec(ctx)
		require.NoError(t, err)
	}

	for _, project := range data.projects {
		if project.DefaultInstanceID == nil {
			continue
		}

		_, err = db.NewRaw(
			"UPDATE projects SET default_instance_id = ?0 WHERE id = ?1", project.DefaultInstanceID, project.ID,
		).Exec(ctx)
		require.NoError(t, err)
	}

	if len(data.links) > 0 {
		_, err = db.NewInsert().Model(&data.links).Returning("NULL").Exec(ctx)
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

// requireStoredBytes checks that the stored bytes of every project match a fresh sum over its rows.
func requireStoredBytes(ctx context.Context, t *testing.T) {
	t.Helper()

	db, err := postgres.GetContext(ctx)
	require.NoError(t, err)

	var counts []*storedBytes

	err = db.NewRaw(storedBytesQuery).Scan(ctx, &counts)
	require.NoError(t, err)

	for _, count := range counts {
		require.Equal(t, count.Bytes, count.Stored, "stored bytes of project %s", count.ProjectID)
	}
}

// projectInstances reads the instances of a project with their documents, oldest first.
func projectInstances(ctx context.Context, t *testing.T, id uuid.UUID) []*dao.Instance {
	t.Helper()

	db, err := postgres.GetContext(ctx)
	require.NoError(t, err)

	instances := []*dao.Instance{}

	err = db.NewSelect().Model(&instances).Where("project_id = ?", id).Order("id").Scan(ctx)
	require.NoError(t, err)

	return instances
}

// projectDefault reads the default instance of a project, nil when it has none or no project has the ID.
func projectDefault(ctx context.Context, t *testing.T, id uuid.UUID) *uuid.UUID {
	t.Helper()

	project, err := dao.NewPgProjectSelect().Exec(ctx, &dao.ProjectSelectRequest{ID: id})
	if errors.Is(err, dao.ErrProjectNotFound) {
		return nil
	}

	require.NoError(t, err)

	return project.DefaultInstanceID
}

// storedLinks reads the links of a project as it stores them, nil when it never wrote them.
func storedLinks(ctx context.Context, t *testing.T, id uuid.UUID) *dao.ProjectLinks {
	t.Helper()

	links, err := dao.NewPgProjectLinksSelect().Exec(ctx, &dao.ProjectLinksSelectRequest{ProjectID: id})
	if errors.Is(err, dao.ErrProjectLinksNotFound) {
		return nil
	}

	require.NoError(t, err)

	return links
}

// linksCount counts the links rows of a project: 1 once it wrote its links, 0 before that and after its delete.
func linksCount(ctx context.Context, t *testing.T, id uuid.UUID) int64 {
	t.Helper()

	db, err := postgres.GetContext(ctx)
	require.NoError(t, err)

	count, err := db.NewSelect().Model((*dao.ProjectLinks)(nil)).Where("project_id = ?", id).Count(ctx)
	require.NoError(t, err)

	return count
}
