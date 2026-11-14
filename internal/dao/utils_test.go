package dao_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/dao"
)

var (
	ownerID      = uuid.MustParse("00000000-0000-0000-0000-0000000000a1")
	otherOwnerID = uuid.MustParse("00000000-0000-0000-0000-0000000000a2")

	projectID      = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	otherProjectID = uuid.MustParse("00000000-0000-0000-0000-000000000002")

	beatID1     = uuid.MustParse("00000000-0000-0000-0000-000000000031")
	beatID2     = uuid.MustParse("00000000-0000-0000-0000-000000000032")
	beatID3     = uuid.MustParse("00000000-0000-0000-0000-000000000033")
	otherBeatID = uuid.MustParse("00000000-0000-0000-0000-000000000039")

	sceneID1     = uuid.MustParse("00000000-0000-0000-0000-000000000011")
	sceneID2     = uuid.MustParse("00000000-0000-0000-0000-000000000012")
	sceneID3     = uuid.MustParse("00000000-0000-0000-0000-000000000013")
	sceneID4     = uuid.MustParse("00000000-0000-0000-0000-000000000016")
	otherSceneID = uuid.MustParse("00000000-0000-0000-0000-000000000019")

	manuscriptLayerID = uuid.MustParse("00000000-0000-0000-0000-000000000021")
	notesLayerID      = uuid.MustParse("00000000-0000-0000-0000-000000000022")
	otherLayerID      = uuid.MustParse("00000000-0000-0000-0000-000000000029")

	// absentID names no row of the fixtures.
	absentID = uuid.MustParse("00000000-0000-0000-0000-0000000000ff")

	// beatColor is the color of every fixture beat.
	beatColor = "#4a90d9"

	createdAt = time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt = time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)
	now       = time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC)
)

// pageBytes bounds the bytes of a list page above what any fixture holds.
const pageBytes = 1 << 20

// storedBytesQuery reads, for each project, the count of the bytes its rows hold, and a fresh sum of them as
// the migration that adds the count backfills it.
const storedBytesQuery = `
SELECT projects.id AS project_id, projects.stored_bytes AS stored, coalesce(sum(held.bytes), 0)::bigint AS bytes
FROM projects
LEFT JOIN (
  SELECT project_id, bytes FROM beats
  UNION ALL SELECT project_id, bytes FROM scenes
  UNION ALL SELECT project_id, bytes FROM layers
  UNION ALL SELECT project_id, bytes FROM cells
) AS held ON held.project_id = projects.id
GROUP BY projects.id`

// storedBytes is the count of the bytes a project's rows hold, beside a fresh sum of them.
type storedBytes struct {
	ProjectID uuid.UUID `bun:"project_id,type:uuid"`
	// Stored is the count the writes keep.
	Stored int64 `bun:"stored"`
	// Bytes is the fresh sum.
	Bytes int64 `bun:"bytes"`
}

// fixtures is the data a DAO test seeds before it runs the operation under test.
type fixtures struct {
	projects []*dao.Project
	logos    []*dao.ProjectLogo
	beats    []*dao.Beat
	scenes   []*dao.Scene
	layers   []*dao.Layer
	cells    []*dao.Cell
}

// composerFixtures returns a project of three beats and two layers beside a project of another owner.
// The first beat holds three scenes, the second one, and the third none. The first scene holds the
// cells. Each call returns fresh values, so a case may change them.
func composerFixtures() fixtures {
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
		beats: []*dao.Beat{
			{ID: beatID1, ProjectID: projectID, Position: 0, Title: "Ouverture", Color: beatColor, UpdatedAt: updatedAt},
			{ID: beatID2, ProjectID: projectID, Position: 1, Title: "Le thème", Color: beatColor, UpdatedAt: updatedAt},
			{ID: beatID3, ProjectID: projectID, Position: 2, Title: "Le déclencheur", Color: beatColor, UpdatedAt: updatedAt},
			{ID: otherBeatID, ProjectID: otherProjectID, Position: 0, Title: "Other", Color: beatColor, UpdatedAt: updatedAt},
		},
		scenes: []*dao.Scene{
			{ID: sceneID1, ProjectID: projectID, BeatID: beatID1, Position: 0, Title: "La fenêtre", UpdatedAt: updatedAt},
			{ID: sceneID2, ProjectID: projectID, BeatID: beatID1, Position: 1, Title: "La rue", UpdatedAt: updatedAt},
			{ID: sceneID3, ProjectID: projectID, BeatID: beatID1, Position: 2, Title: "La porte", UpdatedAt: updatedAt},
			{ID: sceneID4, ProjectID: projectID, BeatID: beatID2, Position: 0, Title: "Le retour", UpdatedAt: updatedAt},
			{
				ID: otherSceneID, ProjectID: otherProjectID, BeatID: otherBeatID, Position: 0, Title: "Other",
				UpdatedAt: updatedAt,
			},
		},
		layers: []*dao.Layer{
			{
				ID: manuscriptLayerID, ProjectID: projectID, Position: 0, Kind: "manuscript", Name: "Manuscript",
				UpdatedAt: updatedAt,
			},
			{
				ID: notesLayerID, ProjectID: projectID, Position: 1, Kind: "notes", Name: "Characters",
				Description: "Who appears", UpdatedAt: updatedAt,
			},
			{
				ID: otherLayerID, ProjectID: otherProjectID, Position: 0, Kind: "manuscript", Name: "Manuscript",
				UpdatedAt: updatedAt,
			},
		},
		cells: []*dao.Cell{
			{
				ProjectID: projectID, SceneID: sceneID1, LayerID: manuscriptLayerID, Content: json.RawMessage(`{"blocks": []}`),
				UpdatedAt: updatedAt,
			},
			{
				ProjectID: projectID, SceneID: sceneID1, LayerID: notesLayerID, Content: json.RawMessage(`{"text": "Marie"}`),
				UpdatedAt: updatedAt,
			},
		},
	}
}

// logoFixtures returns the composer fixtures with a logo on the test project. The logo and the project carry
// the logo's checksum, as a read returns them.
func logoFixtures() fixtures {
	data := composerFixtures()
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

	return lo.ToPtr(hex.EncodeToString(sum[:]))
}

// fillerID derives a stable ID from a filler row's position.
func fillerID(kind string, position int) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(kind+strconv.Itoa(position)))
}

// fillerBeats returns the beats that fill the test project up to its last position, 199.
func fillerBeats() []*dao.Beat {
	beats := make([]*dao.Beat, 0, 197)
	for position := 3; position < 200; position++ {
		beats = append(beats, &dao.Beat{
			ID: fillerID("beat", position), ProjectID: projectID, Position: position, Title: "Filler",
			Color: beatColor, UpdatedAt: updatedAt,
		})
	}

	return beats
}

// fillerScenes returns the scenes that fill the first beat up to its last position, 9.
func fillerScenes() []*dao.Scene {
	scenes := make([]*dao.Scene, 0, 7)
	for position := 3; position < 10; position++ {
		scenes = append(scenes, &dao.Scene{
			ID: fillerID("scene", position), ProjectID: projectID, BeatID: beatID1, Position: position,
			Title: "Filler", UpdatedAt: updatedAt,
		})
	}

	return scenes
}

// fillerLayers returns the layers that fill the test project up to its last position, 999.
func fillerLayers() []*dao.Layer {
	layers := make([]*dao.Layer, 0, 998)
	for position := 2; position < 1000; position++ {
		layers = append(layers, &dao.Layer{
			ID: fillerID("layer", position), ProjectID: projectID, Position: position, Kind: "notes", Name: "Filler",
			UpdatedAt: updatedAt,
		})
	}

	return layers
}

// insertFixtures seeds data in the order its foreign keys need.
func insertFixtures(ctx context.Context, t *testing.T, data fixtures) {
	t.Helper()

	db, err := postgres.GetContext(ctx)
	require.NoError(t, err)

	if len(data.projects) > 0 {
		_, err = db.NewInsert().Model(&data.projects).Exec(ctx)
		require.NoError(t, err)
	}

	if len(data.logos) > 0 {
		_, err = db.NewInsert().Model(&data.logos).Exec(ctx)
		require.NoError(t, err)
	}

	if len(data.beats) > 0 {
		_, err = db.NewInsert().Model(&data.beats).Exec(ctx)
		require.NoError(t, err)
	}

	if len(data.scenes) > 0 {
		_, err = db.NewInsert().Model(&data.scenes).Exec(ctx)
		require.NoError(t, err)
	}

	if len(data.layers) > 0 {
		_, err = db.NewInsert().Model(&data.layers).Exec(ctx)
		require.NoError(t, err)
	}

	if len(data.cells) > 0 {
		_, err = db.NewInsert().Model(&data.cells).Exec(ctx)
		require.NoError(t, err)
	}

	// The inserts skip the writes that keep the stored bytes, so the count starts from a fresh sum.
	_, err = db.NewRaw(
		"UPDATE projects SET stored_bytes = fresh.bytes FROM (" + storedBytesQuery + ") AS fresh " +
			"WHERE projects.id = fresh.project_id",
	).Exec(ctx)
	require.NoError(t, err)
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

// requireCellsEqual compares cells field by field, and their content as JSON, since PostgreSQL
// normalizes the spacing of a jsonb value.
func requireCellsEqual(t *testing.T, expect, actual []*dao.Cell) {
	t.Helper()

	require.Len(t, actual, len(expect))

	for i := range expect {
		require.Equal(t, expect[i].ProjectID, actual[i].ProjectID)
		require.Equal(t, expect[i].SceneID, actual[i].SceneID)
		require.Equal(t, expect[i].LayerID, actual[i].LayerID)
		require.Equal(t, expect[i].UpdatedAt, actual[i].UpdatedAt)
		require.JSONEq(t, string(expect[i].Content), string(actual[i].Content))
		require.Equal(t, expect[i].Recap, actual[i].Recap)
		require.Equal(t, expect[i].RecapCellUpdatedAt, actual[i].RecapCellUpdatedAt)
	}
}

// beatAt returns a copy of a beat, moved to position.
func beatAt(beat *dao.Beat, position int) *dao.Beat {
	moved := *beat
	moved.Position = position

	return &moved
}

// sceneAt returns a copy of a scene, moved to position.
func sceneAt(scene *dao.Scene, position int) *dao.Scene {
	moved := *scene
	moved.Position = position

	return &moved
}

// layerAt returns a copy of a layer, moved to position.
func layerAt(layer *dao.Layer, position int) *dao.Layer {
	moved := *layer
	moved.Position = position

	return &moved
}

// sceneIn returns a copy of a scene, moved to position in beat.
func sceneIn(scene *dao.Scene, beat uuid.UUID, position int) *dao.Scene {
	moved := *scene
	moved.BeatID = beat
	moved.Position = position

	return &moved
}

// listScenes reads every scene of a project, in narrative order.
func listScenes(ctx context.Context, t *testing.T, project uuid.UUID) []*dao.Scene {
	t.Helper()

	request := &dao.SceneListRequest{ProjectID: project, Limit: 100, MaxBytes: pageBytes}

	rows, err := dao.NewPgSceneList().Exec(ctx, request)
	require.NoError(t, err)

	return lo.Map(rows, func(row *dao.SceneListRow, _ int) *dao.Scene { return &row.Scene })
}

// listBeats reads every beat of a project, in narrative order.
func listBeats(ctx context.Context, t *testing.T, project uuid.UUID) []*dao.Beat {
	t.Helper()

	request := &dao.BeatListRequest{ProjectID: project, Limit: 100, MaxBytes: pageBytes}

	rows, err := dao.NewPgBeatList().Exec(ctx, request)
	require.NoError(t, err)

	return lo.Map(rows, func(row *dao.BeatListRow, _ int) *dao.Beat { return &row.Beat })
}

// beatScenes reads every scene of a beat, in order.
func beatScenes(ctx context.Context, t *testing.T, beat uuid.UUID) []*dao.Scene {
	t.Helper()

	return lo.Filter(listScenes(ctx, t, projectID), func(scene *dao.Scene, _ int) bool { return scene.BeatID == beat })
}

// listLayers reads every layer of a project, in order.
func listLayers(ctx context.Context, t *testing.T, project uuid.UUID) []*dao.Layer {
	t.Helper()

	request := &dao.LayerListRequest{ProjectID: project, Limit: 100, MaxBytes: pageBytes}

	rows, err := dao.NewPgLayerList().Exec(ctx, request)
	require.NoError(t, err)

	return lo.Map(rows, func(row *dao.LayerListRow, _ int) *dao.Layer { return &row.Layer })
}

// listCells reads every cell of a project, in order.
func listCells(ctx context.Context, t *testing.T, project uuid.UUID) []*dao.Cell {
	t.Helper()

	request := &dao.CellListRequest{ProjectID: project, Limit: 100, MaxBytes: pageBytes}

	rows, err := dao.NewPgCellList().Exec(ctx, request)
	require.NoError(t, err)

	return lo.Map(rows, func(row *dao.CellListRow, _ int) *dao.Cell { return &row.Cell })
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
