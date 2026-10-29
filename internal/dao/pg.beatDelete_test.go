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

func TestPgBeatDelete(t *testing.T) {
	t.Parallel()

	beats := composerFixtures().beats

	testCases := []struct {
		name string

		request *dao.BeatDeleteRequest

		expect      *dao.Beat
		expectBeats []*dao.Beat
		expectCells []*dao.Cell
		expectErr   error
	}{
		{
			name: "Success",

			request: &dao.BeatDeleteRequest{ProjectID: projectID, ID: beatID1},

			expect:      beats[0],
			expectBeats: []*dao.Beat{beatAt(beats[1], 0), beatAt(beats[2], 1)},
			expectCells: []*dao.Cell{},
		},
		{
			name: "Success/Middle",

			request: &dao.BeatDeleteRequest{ProjectID: projectID, ID: beatID2},

			expect:      beats[1],
			expectBeats: []*dao.Beat{beats[0], beatAt(beats[2], 1)},
			expectCells: composerFixtures().cells,
		},
		{
			name: "Error/NotFound",

			request: &dao.BeatDeleteRequest{ProjectID: projectID, ID: absentID},

			expectBeats: beats[:3],
			expectCells: composerFixtures().cells,
			expectErr:   dao.ErrBeatDeleteNotFound,
		},
		{
			name: "Error/OtherProject",

			request: &dao.BeatDeleteRequest{ProjectID: projectID, ID: otherBeatID},

			expectBeats: beats[:3],
			expectCells: composerFixtures().cells,
			expectErr:   dao.ErrBeatDeleteNotFound,
		},
	}

	daoBeatDelete := dao.NewPgBeatDelete()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoBeatDelete.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)

				// The beats after the deleted one move up, and its cells go with it.
				require.Equal(t, testCase.expectBeats, listBeats(ctx, t, projectID))
				requireCellsEqual(t, testCase.expectCells, listCells(ctx, t, projectID))
			})
		})
	}
}
