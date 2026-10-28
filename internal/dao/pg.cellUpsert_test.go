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

			request: &dao.CellUpsertRequest{
				ProjectID: projectID, SceneID: sceneID2, LayerID: notesLayerID, Content: content, Now: now,
			},

			expect: &dao.Cell{
				ProjectID: projectID, SceneID: sceneID2, LayerID: notesLayerID, Content: content, UpdatedAt: now,
			},
		},
		{
			name: "Success/Update",

			request: &dao.CellUpsertRequest{
				ProjectID: projectID, SceneID: sceneID1, LayerID: notesLayerID, Content: content, Now: now,
			},

			expect: &dao.Cell{
				ProjectID: projectID, SceneID: sceneID1, LayerID: notesLayerID, Content: content, UpdatedAt: now,
			},
		},
		{
			name: "Error/UnknownScene",

			request: &dao.CellUpsertRequest{
				ProjectID: projectID, SceneID: absentID, LayerID: notesLayerID, Content: content, Now: now,
			},

			expectErr: dao.ErrCellUpsertSceneNotFound,
		},
		{
			name: "Error/OtherProjectScene",

			request: &dao.CellUpsertRequest{
				ProjectID: projectID, SceneID: otherSceneID, LayerID: notesLayerID, Content: content, Now: now,
			},

			expectErr: dao.ErrCellUpsertSceneNotFound,
		},
		{
			name: "Error/OtherProjectLayer",

			request: &dao.CellUpsertRequest{
				ProjectID: projectID, SceneID: sceneID2, LayerID: otherLayerID, Content: content, Now: now,
			},

			expectErr: dao.ErrCellUpsertLayerNotFound,
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

				if testCase.expectErr != nil {
					return
				}

				requireCellsEqual(t, []*dao.Cell{testCase.expect}, []*dao.Cell{result})
			})
		})
	}
}
