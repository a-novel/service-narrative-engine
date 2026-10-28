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

func TestPgLayerList(t *testing.T) {
	t.Parallel()

	layers := composerFixtures().layers

	row := func(layer *dao.Layer) *dao.LayerListRow { return &dao.LayerListRow{Layer: *layer, Total: 2} }

	testCases := []struct {
		name string

		request *dao.LayerListRequest

		expect    []*dao.LayerListRow
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.LayerListRequest{ProjectID: projectID, Limit: 10},

			expect: []*dao.LayerListRow{row(layers[0]), row(layers[1])},
		},
		{
			name: "Success/Page",

			request: &dao.LayerListRequest{ProjectID: projectID, Limit: 1, Offset: 1},

			expect: []*dao.LayerListRow{row(layers[1])},
		},
		{
			name: "Success/PastEnd",

			request: &dao.LayerListRequest{ProjectID: projectID, Limit: 10, Offset: 2},
		},
	}

	daoLayerList := dao.NewPgLayerList()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoLayerList.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
