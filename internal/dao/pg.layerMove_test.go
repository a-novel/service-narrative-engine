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

func TestPgLayerMove(t *testing.T) {
	t.Parallel()

	layers := composerFixtures().layers

	testCases := []struct {
		name string

		request *dao.LayerMoveRequest

		expect       *dao.Layer
		expectLayers []*dao.Layer
		expectErr    error
	}{
		{
			name: "Success/Up",

			request: &dao.LayerMoveRequest{ID: notesLayerID, To: 0},

			expect:       layerAt(layers[1], 0),
			expectLayers: []*dao.Layer{layerAt(layers[1], 0), layerAt(layers[0], 1)},
		},
		{
			name: "Success/Down",

			request: &dao.LayerMoveRequest{ID: manuscriptLayerID, To: 1},

			expect:       layerAt(layers[0], 1),
			expectLayers: []*dao.Layer{layerAt(layers[1], 0), layerAt(layers[0], 1)},
		},
		{
			name: "Error/NotFound",

			request: &dao.LayerMoveRequest{ID: absentID, To: 0},

			expectErr: dao.ErrLayerMoveNotFound,
		},
	}

	daoLayerMove := dao.NewPgLayerMove()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoLayerMove.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)

				if testCase.expectErr == nil {
					require.Equal(t, testCase.expectLayers, listLayers(ctx, t, projectID))
				}
			})
		})
	}
}
