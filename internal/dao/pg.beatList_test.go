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

func TestPgBeatList(t *testing.T) {
	t.Parallel()

	beats := composerFixtures().beats

	row := func(beat *dao.Beat) *dao.BeatListRow { return &dao.BeatListRow{Beat: *beat, Total: 3} }

	testCases := []struct {
		name string

		request *dao.BeatListRequest

		expect    []*dao.BeatListRow
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.BeatListRequest{ProjectID: projectID, Limit: 10},

			expect: []*dao.BeatListRow{row(beats[0]), row(beats[1]), row(beats[2])},
		},
		{
			name: "Success/Page",

			request: &dao.BeatListRequest{ProjectID: projectID, Limit: 1, Offset: 1},

			expect: []*dao.BeatListRow{row(beats[1])},
		},
		{
			name: "Success/PastEnd",

			request: &dao.BeatListRequest{ProjectID: projectID, Limit: 10, Offset: 3},
		},
	}

	daoBeatList := dao.NewPgBeatList()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoBeatList.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
