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

func TestPgBeatsUpdate(t *testing.T) {
	t.Parallel()

	beats := composerFixtures().beats

	// updateBeat2 and updateBeat3 rewrite the second and third beats, which become beat2 and beat3.
	updateBeat2 := dao.BeatsUpdateItem{
		ID: beatID2, Title: "Le mensonge", Color: "#e0457b", Description: "Paul ment.", Prompt: "Garder le suspense.",
	}
	updateBeat3 := dao.BeatsUpdateItem{ID: beatID3, Title: "Le départ", Color: "#2bb673"}
	beat2 := &dao.Beat{
		ID: beatID2, ProjectID: projectID, Position: 1, Title: "Le mensonge", Color: "#e0457b",
		Description: "Paul ment.", Prompt: "Garder le suspense.", UpdatedAt: now,
	}
	beat3 := &dao.Beat{
		ID: beatID3, ProjectID: projectID, Position: 2, Title: "Le départ", Color: "#2bb673", UpdatedAt: now,
	}

	testCases := []struct {
		name string

		request *dao.BeatsUpdateRequest

		expect []*dao.Beat
		// expectBeats is the project's order after the update.
		expectBeats []*dao.Beat
	}{
		{
			name: "Success",

			request: &dao.BeatsUpdateRequest{ProjectID: projectID, Beats: []dao.BeatsUpdateItem{updateBeat2}, Now: now},

			expect:      []*dao.Beat{beat2},
			expectBeats: []*dao.Beat{beats[0], beat2, beats[2]},
		},
		{
			name: "Success/Batch",

			request: &dao.BeatsUpdateRequest{
				ProjectID: projectID, Beats: []dao.BeatsUpdateItem{updateBeat2, updateBeat3}, Now: now,
			},

			expect:      []*dao.Beat{beat2, beat3},
			expectBeats: []*dao.Beat{beats[0], beat2, beat3},
		},
		{
			name: "Success/NotFound",

			request: &dao.BeatsUpdateRequest{
				ProjectID: projectID,
				Beats:     []dao.BeatsUpdateItem{updateBeat2, {ID: absentID, Title: "Le départ", Color: "#2bb673"}},
				Now:       now,
			},

			expectBeats: beats[:3],
		},
		{
			name: "Success/OtherProject",

			request: &dao.BeatsUpdateRequest{
				ProjectID: projectID,
				Beats:     []dao.BeatsUpdateItem{updateBeat2, {ID: otherBeatID, Title: "Le départ", Color: "#2bb673"}},
				Now:       now,
			},

			expectBeats: beats[:3],
		},
	}

	daoBeatsUpdate := dao.NewPgBeatsUpdate()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoBeatsUpdate.Exec(ctx, testCase.request)
				require.NoError(t, err)
				require.ElementsMatch(t, testCase.expect, result)

				// An ID the project does not hold leaves every beat unchanged, the valid ones included.
				require.Equal(t, testCase.expectBeats, listBeats(ctx, t, projectID))
			})
		})
	}
}
