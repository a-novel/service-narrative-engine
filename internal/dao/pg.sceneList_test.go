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

func TestPgSceneList(t *testing.T) {
	t.Parallel()

	scenes := composerFixtures().scenes

	row := func(scene *dao.Scene) *dao.SceneListRow { return &dao.SceneListRow{Scene: *scene, Total: 4} }

	testCases := []struct {
		name string

		request *dao.SceneListRequest

		expect    []*dao.SceneListRow
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.SceneListRequest{ProjectID: projectID, Limit: 10},

			// The second beat's scene comes after the first beat's, though its position in its beat is 0.
			expect: []*dao.SceneListRow{row(scenes[0]), row(scenes[1]), row(scenes[2]), row(scenes[3])},
		},
		{
			name: "Success/Page",

			request: &dao.SceneListRequest{ProjectID: projectID, Limit: 1, Offset: 1},

			expect: []*dao.SceneListRow{row(scenes[1])},
		},
		{
			name: "Success/PastEnd",

			request: &dao.SceneListRequest{ProjectID: projectID, Limit: 10, Offset: 4},
		},
	}

	daoSceneList := dao.NewPgSceneList()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoSceneList.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
