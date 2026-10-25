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

func TestPgLayerReorder(t *testing.T) {
	t.Parallel()

	moved := func(layer *dao.Layer, position int) *dao.Layer {
		layer.Position = position

		return layer
	}

	testCases := []struct {
		name string

		request *dao.LayerReorderRequest

		expect    []*dao.Layer
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.LayerReorderRequest{ProjectID: projectID, IDs: []uuid.UUID{notesLayerID, manuscriptLayerID}},

			expect: []*dao.Layer{moved(composerFixtures().layers[1], 0), moved(composerFixtures().layers[0], 1)},
		},
		{
			name: "Success/OtherProjectIgnored",

			request: &dao.LayerReorderRequest{ProjectID: projectID, IDs: []uuid.UUID{manuscriptLayerID, otherLayerID}},

			expect: []*dao.Layer{composerFixtures().layers[0]},
		},
	}

	daoLayerReorder := dao.NewPgLayerReorder()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoLayerReorder.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)

				slices.SortFunc(result, func(a, b *dao.Layer) int { return cmp.Compare(a.Position, b.Position) })
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
