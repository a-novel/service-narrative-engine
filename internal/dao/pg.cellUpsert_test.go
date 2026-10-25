package dao_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgCellUpsert(t *testing.T) {
	t.Parallel()

	content := json.RawMessage(`{"text": "Marie, Paul"}`)

	testCases := []struct {
		name string

		request *dao.CellUpsertRequest

		expect    *dao.Cell
		expectErr error
	}{
		{
			name: "Success/Create",

			request: &dao.CellUpsertRequest{SceneID: sceneID2, LayerID: notesLayerID, Revision: 0, Content: content, Now: now},

			expect: &dao.Cell{SceneID: sceneID2, LayerID: notesLayerID, Content: content, Revision: 1, UpdatedAt: now},
		},
		{
			name: "Success/Update",

			request: &dao.CellUpsertRequest{SceneID: sceneID1, LayerID: notesLayerID, Revision: 2, Content: content, Now: now},

			expect: &dao.Cell{SceneID: sceneID1, LayerID: notesLayerID, Content: content, Revision: 3, UpdatedAt: now},
		},
		{
			name: "Error/StaleRevision",

			request: &dao.CellUpsertRequest{SceneID: sceneID1, LayerID: notesLayerID, Revision: 1, Content: content, Now: now},

			expectErr: dao.ErrCellUpsertNotFound,
		},
		{
			name: "Error/CreateExisting",

			request: &dao.CellUpsertRequest{SceneID: sceneID1, LayerID: notesLayerID, Revision: 0, Content: content, Now: now},

			expectErr: dao.ErrCellUpsertNotFound,
		},
		{
			name: "Error/UpdateMissing",

			request: &dao.CellUpsertRequest{SceneID: sceneID2, LayerID: notesLayerID, Revision: 1, Content: content, Now: now},

			expectErr: dao.ErrCellUpsertNotFound,
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
				require.ErrorIs(t, err, testCase.expectErr)

				if testCase.expect == nil {
					require.Nil(t, result)

					return
				}

				requireCellsEqual(t, []*dao.Cell{testCase.expect}, []*dao.Cell{result})
			})
		})
	}
}
