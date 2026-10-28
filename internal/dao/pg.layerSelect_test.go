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

func TestPgLayerSelect(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *dao.LayerSelectRequest

		expect    *dao.Layer
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.LayerSelectRequest{ProjectID: projectID, ID: notesLayerID},

			expect: composerFixtures().layers[1],
		},
		{
			name: "Error/NotFound",

			request: &dao.LayerSelectRequest{ProjectID: projectID, ID: absentID},

			expectErr: dao.ErrLayerSelectNotFound,
		},
		{
			name: "Error/OtherProject",

			request: &dao.LayerSelectRequest{ProjectID: projectID, ID: otherLayerID},

			expectErr: dao.ErrLayerSelectNotFound,
		},
	}

	daoLayerSelect := dao.NewPgLayerSelect()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoLayerSelect.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
