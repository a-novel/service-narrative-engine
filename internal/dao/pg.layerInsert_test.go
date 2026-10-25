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

		expect       *dao.Layer
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
				Description: "Where it happens", Revision: 1, UpdatedAt: now,
			},
		},
		{
			name: "Error/SecondManuscript",

			request: &dao.LayerInsertRequest{
				ID: newLayerID, ProjectID: projectID, Position: 2, Kind: "manuscript", Name: "Manuscript", Now: now,
			},

			expectFailed: true,
		},
		{
			name: "Error/PositionTaken",

			request: &dao.LayerInsertRequest{
				ID: newLayerID, ProjectID: projectID, Position: 1, Kind: "notes", Name: "Places", Now: now,
			},

			expectFailed: true,
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

				require.NoError(t, err)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
