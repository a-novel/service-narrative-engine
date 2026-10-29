package dao_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgBeatInsert(t *testing.T) {
	t.Parallel()

	newBeatID := uuid.MustParse("00000000-0000-0000-0000-000000000034")

	testCases := []struct {
		name string

		request *dao.BeatInsertRequest

		expect *dao.Beat
		// expectBeats is the project's order after the insert, when the case checks it.
		expectBeats []*dao.Beat
		// full fills the project up to its last position before the insert.
		full      bool
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.BeatInsertRequest{
				ID: newBeatID, ProjectID: projectID, Position: 3, Title: "Le débat", Color: "#e0457b",
				Description: "Marie hésite.", Prompt: "Lent.", Now: now,
			},

			expect: &dao.Beat{
				ID: newBeatID, ProjectID: projectID, Position: 3, Title: "Le débat", Color: "#e0457b",
				Description: "Marie hésite.", Prompt: "Lent.", UpdatedAt: now,
			},
		},
		{
			name: "Success/PastTheEnd",

			request: &dao.BeatInsertRequest{
				ID: newBeatID, ProjectID: projectID, Position: 199, Title: "Le débat", Color: "#e0457b", Now: now,
			},

			expect: &dao.Beat{
				ID: newBeatID, ProjectID: projectID, Position: 3, Title: "Le débat", Color: "#e0457b", UpdatedAt: now,
			},
		},
		{
			name: "Error/ProjectFull",

			request: &dao.BeatInsertRequest{
				ID: newBeatID, ProjectID: projectID, Position: 0, Title: "Le débat", Color: "#e0457b", Now: now,
			},

			full:      true,
			expectErr: dao.ErrBeatInsertFull,
		},
		{
			name: "Success/Shift",

			request: &dao.BeatInsertRequest{
				ID: newBeatID, ProjectID: projectID, Position: 1, Title: "Le débat", Color: "#e0457b", Now: now,
			},

			expect: &dao.Beat{
				ID: newBeatID, ProjectID: projectID, Position: 1, Title: "Le débat", Color: "#e0457b", UpdatedAt: now,
			},
			expectBeats: []*dao.Beat{
				composerFixtures().beats[0],
				{ID: newBeatID, ProjectID: projectID, Position: 1, Title: "Le débat", Color: "#e0457b", UpdatedAt: now},
				beatAt(composerFixtures().beats[1], 2),
				beatAt(composerFixtures().beats[2], 3),
			},
		},
	}

	daoBeatInsert := dao.NewPgBeatInsert()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				data := composerFixtures()
				if testCase.full {
					data.beats = append(data.beats, fillerBeats()...)
				}

				insertFixtures(ctx, t, data)

				result, err := daoBeatInsert.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)

				if testCase.expectBeats != nil {
					require.Equal(t, testCase.expectBeats, listBeats(ctx, t, projectID))
				}
			})
		})
	}
}
