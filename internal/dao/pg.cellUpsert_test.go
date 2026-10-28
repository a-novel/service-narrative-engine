package dao_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgCellUpsert(t *testing.T) {
	t.Parallel()

	content := json.RawMessage(`{"text": "Marie, Paul"}`)
	// unknownSceneID names no scene, so a cell for it breaks the foreign key.
	unknownSceneID := uuid.MustParse("00000000-0000-0000-0000-000000000018")

	testCases := []struct {
		name string

		request *dao.CellUpsertRequest

		expect       *dao.Cell
		expectFailed bool
	}{
		{
			name: "Success/Create",

			request: &dao.CellUpsertRequest{SceneID: sceneID2, LayerID: notesLayerID, Content: content, Now: now},

			expect: &dao.Cell{SceneID: sceneID2, LayerID: notesLayerID, Content: content, UpdatedAt: now},
		},
		{
			name: "Success/Update",

			request: &dao.CellUpsertRequest{SceneID: sceneID1, LayerID: notesLayerID, Content: content, Now: now},

			expect: &dao.Cell{SceneID: sceneID1, LayerID: notesLayerID, Content: content, UpdatedAt: now},
		},
		{
			name: "Error/UnknownScene",

			request: &dao.CellUpsertRequest{SceneID: unknownSceneID, LayerID: notesLayerID, Content: content, Now: now},

			expectFailed: true,
		},
	}

	daoCellUpsert := dao.NewPgCellUpsert()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoCellUpsert.Exec(ctx, testCase.request)
				if testCase.expectFailed {
					require.Error(t, err)

					return
				}

				require.NoError(t, err)

				requireCellsEqual(t, []*dao.Cell{testCase.expect}, []*dao.Cell{result})
			})
		})
	}
}
