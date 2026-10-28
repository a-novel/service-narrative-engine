package dao_test

import (
	"context"
	"encoding/json"
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

	sceneID1     = uuid.MustParse("00000000-0000-0000-0000-000000000011")
	sceneID2     = uuid.MustParse("00000000-0000-0000-0000-000000000012")
	sceneID3     = uuid.MustParse("00000000-0000-0000-0000-000000000013")
	otherSceneID = uuid.MustParse("00000000-0000-0000-0000-000000000019")

	manuscriptLayerID = uuid.MustParse("00000000-0000-0000-0000-000000000021")
	notesLayerID      = uuid.MustParse("00000000-0000-0000-0000-000000000022")
	otherLayerID      = uuid.MustParse("00000000-0000-0000-0000-000000000029")

	// absentID names no row of the fixtures.
	absentID = uuid.MustParse("00000000-0000-0000-0000-0000000000ff")

	// sceneColor is the color of every fixture scene.
	sceneColor = "#4a90d9"

	createdAt = time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt = time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)
	now       = time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC)
)

// fixtures is the data a DAO test seeds before it runs the operation under test.
type fixtures struct {
	projects []*dao.Project
	scenes   []*dao.Scene
	layers   []*dao.Layer
	cells    []*dao.Cell
}

// composerFixtures returns a project of three scenes and two layers, with cells in its first scene,
// beside a project of another owner. Each call returns fresh values, so a case may change them.
func composerFixtures() fixtures {
	return fixtures{
		projects: []*dao.Project{
			{
				ID: projectID, OwnerID: ownerID, Title: "Les Heures claires", Premise: "Marie attend.",
				Language: "fr", CreatedAt: createdAt, UpdatedAt: updatedAt,
			},
			{
				ID: otherProjectID, OwnerID: otherOwnerID, Title: "Other", Language: "en",
				CreatedAt: createdAt, UpdatedAt: updatedAt,
			},
		},
		scenes: []*dao.Scene{
			{ID: sceneID1, ProjectID: projectID, Position: 0, Title: "Ouverture", Color: sceneColor, UpdatedAt: updatedAt},
			{ID: sceneID2, ProjectID: projectID, Position: 1, Title: "Le thème", Color: sceneColor, UpdatedAt: updatedAt},
			{ID: sceneID3, ProjectID: projectID, Position: 2, Title: "Le déclencheur", Color: sceneColor, UpdatedAt: updatedAt},
			{ID: otherSceneID, ProjectID: otherProjectID, Position: 0, Title: "Other", Color: sceneColor, UpdatedAt: updatedAt},
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

// fillerID derives a stable ID from a filler row's position.
func fillerID(kind string, position int) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(kind+strconv.Itoa(position)))
}

// fillerScenes returns the scenes that fill the test project up to its last position, 499.
func fillerScenes() []*dao.Scene {
	scenes := make([]*dao.Scene, 0, 497)
	for position := 3; position < 500; position++ {
		scenes = append(scenes, &dao.Scene{
			ID: fillerID("scene", position), ProjectID: projectID, Position: position, Title: "Filler",
			Color: sceneColor, UpdatedAt: updatedAt,
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
	}
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

// listScenes reads every scene of a project, in narrative order.
func listScenes(ctx context.Context, t *testing.T, project uuid.UUID) []*dao.Scene {
	t.Helper()

	rows, err := dao.NewPgSceneList().Exec(ctx, &dao.SceneListRequest{ProjectID: project, Limit: 100})
	require.NoError(t, err)

	return lo.Map(rows, func(row *dao.SceneListRow, _ int) *dao.Scene { return &row.Scene })
}

// listLayers reads every layer of a project, in order.
func listLayers(ctx context.Context, t *testing.T, project uuid.UUID) []*dao.Layer {
	t.Helper()

	rows, err := dao.NewPgLayerList().Exec(ctx, &dao.LayerListRequest{ProjectID: project, Limit: 100})
	require.NoError(t, err)

	return lo.Map(rows, func(row *dao.LayerListRow, _ int) *dao.Layer { return &row.Layer })
}

// listCells reads every cell of a project, in order.
func listCells(ctx context.Context, t *testing.T, project uuid.UUID) []*dao.Cell {
	t.Helper()

	rows, err := dao.NewPgCellList().Exec(ctx, &dao.CellListRequest{ProjectID: project, Limit: 100})
	require.NoError(t, err)

	return lo.Map(rows, func(row *dao.CellListRow, _ int) *dao.Cell { return &row.Cell })
}
