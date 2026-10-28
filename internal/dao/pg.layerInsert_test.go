package dao_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgLayerInsert(t *testing.T) {
	t.Parallel()

	newLayerID := uuid.MustParse("00000000-0000-0000-0000-000000000023")

	testCases := []struct {
		name string

		request *dao.LayerInsertRequest

		expect *dao.Layer
		// expectLayers is the project's order after the insert, when the case checks it.
		expectLayers []*dao.Layer
		expectErr    error
		expectFailed bool
	}{
		{
			name: "Success",

			request: &dao.LayerInsertRequest{
				ID: newLayerID, ProjectID: projectID, Position: 2, Kind: "notes", Name: "Places",
				Description: "Where it happens", Now: now,
			},

			expect: &dao.Layer{
				ID: newLayerID, ProjectID: projectID, Position: 2, Kind: "notes", Name: "Places",
				Description: "Where it happens", UpdatedAt: now,
			},
		},
		{
			name: "Success/LastPosition",

			request: &dao.LayerInsertRequest{
				ID: newLayerID, ProjectID: projectID, Position: 999, Kind: "notes", Name: "Places", Now: now,
			},

			expect: &dao.Layer{
				ID: newLayerID, ProjectID: projectID, Position: 999, Kind: "notes", Name: "Places", UpdatedAt: now,
			},
		},
		{
			name: "Error/ProjectFull",

			request: &dao.LayerInsertRequest{
				ID: newLayerID, ProjectID: projectID, Position: 1000, Kind: "notes", Name: "Places", Now: now,
			},

			expectErr: dao.ErrLayerInsertFull,
		},
		{
			name: "Error/SecondManuscript",

			request: &dao.LayerInsertRequest{
				ID: newLayerID, ProjectID: projectID, Position: 2, Kind: "manuscript", Name: "Manuscript", Now: now,
			},

			expectFailed: true,
		},
		{
			name: "Success/Shift",

			request: &dao.LayerInsertRequest{
				ID: newLayerID, ProjectID: projectID, Position: 1, Kind: "notes", Name: "Places", Now: now,
			},

			expect: &dao.Layer{
				ID: newLayerID, ProjectID: projectID, Position: 1, Kind: "notes", Name: "Places", UpdatedAt: now,
			},
			expectLayers: []*dao.Layer{
				composerFixtures().layers[0],
				{ID: newLayerID, ProjectID: projectID, Position: 1, Kind: "notes", Name: "Places", UpdatedAt: now},
				layerAt(composerFixtures().layers[1], 2),
			},
		},
	}

	daoLayerInsert := dao.NewPgLayerInsert()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoLayerInsert.Exec(ctx, testCase.request)
				if testCase.expectFailed {
					require.Error(t, err)

					return
				}

				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)

				if testCase.expectLayers != nil {
					require.Equal(t, testCase.expectLayers, listLayers(ctx, t, projectID))
				}
			})
		})
	}
}
