package dao_test

import (
	"context"
	"encoding/json"
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

	sceneID1     = uuid.MustParse("00000000-0000-0000-0000-000000000011")
	sceneID2     = uuid.MustParse("00000000-0000-0000-0000-000000000012")
	sceneID3     = uuid.MustParse("00000000-0000-0000-0000-000000000013")
	otherSceneID = uuid.MustParse("00000000-0000-0000-0000-000000000019")

	manuscriptLayerID = uuid.MustParse("00000000-0000-0000-0000-000000000021")
	notesLayerID      = uuid.MustParse("00000000-0000-0000-0000-000000000022")
	otherLayerID      = uuid.MustParse("00000000-0000-0000-0000-000000000029")

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
				Language: "fr", Revision: 1, CreatedAt: createdAt, UpdatedAt: updatedAt,
			},
			{
				ID: otherProjectID, OwnerID: otherOwnerID, Title: "Other", Language: "en",
				Revision: 1, CreatedAt: createdAt, UpdatedAt: updatedAt,
			},
		},
		scenes: []*dao.Scene{
			{ID: sceneID1, ProjectID: projectID, Position: 0, Title: "Ouverture", Revision: 1, UpdatedAt: updatedAt},
			{ID: sceneID2, ProjectID: projectID, Position: 1, Title: "Le thème", Revision: 1, UpdatedAt: updatedAt},
			{ID: sceneID3, ProjectID: projectID, Position: 2, Title: "Le déclencheur", Revision: 1, UpdatedAt: updatedAt},
			{ID: otherSceneID, ProjectID: otherProjectID, Position: 0, Title: "Other", Revision: 1, UpdatedAt: updatedAt},
		},
		layers: []*dao.Layer{
			{
				ID: manuscriptLayerID, ProjectID: projectID, Position: 0, Kind: "manuscript", Name: "Manuscript",
				Revision: 1, UpdatedAt: updatedAt,
			},
			{
				ID: notesLayerID, ProjectID: projectID, Position: 1, Kind: "notes", Name: "Characters",
				Description: "Who appears", Revision: 1, UpdatedAt: updatedAt,
			},
			{
				ID: otherLayerID, ProjectID: otherProjectID, Position: 0, Kind: "manuscript", Name: "Manuscript",
				Revision: 1, UpdatedAt: updatedAt,
			},
		},
		cells: []*dao.Cell{
			{
				SceneID: sceneID1, LayerID: manuscriptLayerID, Content: json.RawMessage(`{"blocks": []}`),
				Revision: 1, UpdatedAt: updatedAt,
			},
			{
				SceneID: sceneID1, LayerID: notesLayerID, Content: json.RawMessage(`{"text": "Marie"}`),
				Revision: 2, UpdatedAt: updatedAt,
			},
		},
	}
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
		require.Equal(t, expect[i].SceneID, actual[i].SceneID)
		require.Equal(t, expect[i].LayerID, actual[i].LayerID)
		require.Equal(t, expect[i].Revision, actual[i].Revision)
		require.Equal(t, expect[i].UpdatedAt, actual[i].UpdatedAt)
		require.JSONEq(t, string(expect[i].Content), string(actual[i].Content))
	}
}
