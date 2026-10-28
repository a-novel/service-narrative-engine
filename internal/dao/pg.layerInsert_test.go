package dao_test

import (
	"cmp"
	"context"
	"slices"
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

	newLayerID1 := uuid.MustParse("00000000-0000-0000-0000-000000000023")
	newLayerID2 := uuid.MustParse("00000000-0000-0000-0000-000000000024")

	testCases := []struct {
		name string

		request *dao.LayerInsertRequest

		expect []*dao.Layer
		// expectLayers is the project's order after the insert, when the case checks it.
		expectLayers []*dao.Layer
		// full fills the project up to its last position before the insert.
		full      bool
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.LayerInsertRequest{
				ProjectID: projectID,
				Position:  2,
				Layers: []dao.LayerInsertItem{
					{ID: newLayerID1, Kind: "notes", Name: "Places", Description: "Where it happens"},
					{ID: newLayerID2, Kind: "notes", Name: "Objects"},
				},
				Now: now,
			},

			expect: []*dao.Layer{
				{
					ID: newLayerID1, ProjectID: projectID, Position: 2, Kind: "notes", Name: "Places",
					Description: "Where it happens", UpdatedAt: now,
				},
				{ID: newLayerID2, ProjectID: projectID, Position: 3, Kind: "notes", Name: "Objects", UpdatedAt: now},
			},
		},
		{
			name: "Success/PastTheEnd",

			request: &dao.LayerInsertRequest{
				ProjectID: projectID,
				Position:  999,
				Layers:    []dao.LayerInsertItem{{ID: newLayerID1, Kind: "notes", Name: "Places"}},
				Now:       now,
			},

			expect: []*dao.Layer{
				{ID: newLayerID1, ProjectID: projectID, Position: 2, Kind: "notes", Name: "Places", UpdatedAt: now},
			},
		},
		{
			name: "Success/SecondManuscript",

			request: &dao.LayerInsertRequest{
				ProjectID: projectID,
				Position:  2,
				Layers:    []dao.LayerInsertItem{{ID: newLayerID1, Kind: "manuscript", Name: "Draft"}},
				Now:       now,
			},

			expect: []*dao.Layer{
				{ID: newLayerID1, ProjectID: projectID, Position: 2, Kind: "manuscript", Name: "Draft", UpdatedAt: now},
			},
			expectLayers: []*dao.Layer{
				composerFixtures().layers[0],
				composerFixtures().layers[1],
				{ID: newLayerID1, ProjectID: projectID, Position: 2, Kind: "manuscript", Name: "Draft", UpdatedAt: now},
			},
		},
		{
			name: "Error/ProjectFull",

			request: &dao.LayerInsertRequest{
				ProjectID: projectID,
				Position:  0,
				Layers:    []dao.LayerInsertItem{{ID: newLayerID1, Kind: "notes", Name: "Places"}},
				Now:       now,
			},

			full:      true,
			expectErr: dao.ErrLayerInsertFull,
		},
		{
			name: "Success/Shift",

			request: &dao.LayerInsertRequest{
				ProjectID: projectID,
				Position:  1,
				Layers: []dao.LayerInsertItem{
					{ID: newLayerID1, Kind: "notes", Name: "Places"},
					{ID: newLayerID2, Kind: "notes", Name: "Objects"},
				},
				Now: now,
			},

			expect: []*dao.Layer{
				{ID: newLayerID1, ProjectID: projectID, Position: 1, Kind: "notes", Name: "Places", UpdatedAt: now},
				{ID: newLayerID2, ProjectID: projectID, Position: 2, Kind: "notes", Name: "Objects", UpdatedAt: now},
			},
			expectLayers: []*dao.Layer{
				composerFixtures().layers[0],
				{ID: newLayerID1, ProjectID: projectID, Position: 1, Kind: "notes", Name: "Places", UpdatedAt: now},
				{ID: newLayerID2, ProjectID: projectID, Position: 2, Kind: "notes", Name: "Objects", UpdatedAt: now},
				layerAt(composerFixtures().layers[1], 3),
			},
		},
	}

	daoLayerInsert := dao.NewPgLayerInsert()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				data := composerFixtures()
				if testCase.full {
					data.layers = append(data.layers, fillerLayers()...)
				}

				insertFixtures(ctx, t, data)

				result, err := daoLayerInsert.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)

				slices.SortFunc(result, func(a, b *dao.Layer) int { return cmp.Compare(a.Position, b.Position) })
				require.Equal(t, testCase.expect, result)

				if testCase.expectLayers != nil {
					require.Equal(t, testCase.expectLayers, listLayers(ctx, t, projectID))
				}
			})
		})
	}
}
