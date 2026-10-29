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

func TestPgBeatMove(t *testing.T) {
	t.Parallel()

	beats := composerFixtures().beats

	testCases := []struct {
		name string

		request *dao.BeatMoveRequest

		expect      *dao.Beat
		expectBeats []*dao.Beat
		expectErr   error
	}{
		{
			name: "Success/Up",

			request: &dao.BeatMoveRequest{ProjectID: projectID, ID: beatID3, Position: 0},

			expect:      beatAt(beats[2], 0),
			expectBeats: []*dao.Beat{beatAt(beats[2], 0), beatAt(beats[0], 1), beatAt(beats[1], 2)},
		},
		{
			name: "Success/Down",

			request: &dao.BeatMoveRequest{ProjectID: projectID, ID: beatID1, Position: 1},

			expect:      beatAt(beats[0], 1),
			expectBeats: []*dao.Beat{beatAt(beats[1], 0), beatAt(beats[0], 1), beats[2]},
		},
		{
			name: "Success/PastTheEnd",

			request: &dao.BeatMoveRequest{ProjectID: projectID, ID: beatID1, Position: 99},

			expect:      beatAt(beats[0], 2),
			expectBeats: []*dao.Beat{beatAt(beats[1], 0), beatAt(beats[2], 1), beatAt(beats[0], 2)},
		},
		{
			name: "Success/SamePosition",

			request: &dao.BeatMoveRequest{ProjectID: projectID, ID: beatID2, Position: 1},

			expect:      beats[1],
			expectBeats: beats[:3],
		},
		{
			name: "Error/NotFound",

			request: &dao.BeatMoveRequest{ProjectID: projectID, ID: absentID, Position: 0},

			expectErr: dao.ErrBeatMoveNotFound,
		},
		{
			name: "Error/OtherProject",

			request: &dao.BeatMoveRequest{ProjectID: projectID, ID: otherBeatID, Position: 0},

			expectErr: dao.ErrBeatMoveNotFound,
		},
	}

	daoBeatMove := dao.NewPgBeatMove()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoBeatMove.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)

				if testCase.expectErr == nil {
					require.Equal(t, testCase.expectBeats, listBeats(ctx, t, projectID))
				}
			})
		})
	}
}
