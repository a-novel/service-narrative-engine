package dao_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgLayerUpdate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *dao.LayerUpdateRequest

		expect    *dao.Layer
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.LayerUpdateRequest{
				ID: notesLayerID, Name: "Cast", Description: "Who speaks", Now: now,
			},

			expect: &dao.Layer{
				ID: notesLayerID, ProjectID: projectID, Position: 1, Kind: "notes", Name: "Cast",
				Description: "Who speaks", UpdatedAt: now,
			},
		},
		{
			name: "Error/NotFound",

			request: &dao.LayerUpdateRequest{ID: absentID, Name: "Cast", Now: now},

			expectErr: dao.ErrLayerUpdateNotFound,
		},
	}

	daoLayerUpdate := dao.NewPgLayerUpdate()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoLayerUpdate.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
