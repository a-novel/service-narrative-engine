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

func TestPgBeatsInsert(t *testing.T) {
	t.Parallel()

	newBeatID1 := uuid.MustParse("00000000-0000-0000-0000-000000000034")
	newBeatID2 := uuid.MustParse("00000000-0000-0000-0000-000000000035")

	testCases := []struct {
		name string

		request *dao.BeatsInsertRequest

		expect []*dao.Beat
		// expectBeats is the project's order after the insert, when the case checks it.
		expectBeats []*dao.Beat
		// full fills the project up to its last position before the insert.
		full      bool
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.BeatsInsertRequest{
				ProjectID: projectID,
				Position:  3,
				Beats: []dao.BeatsInsertItem{
					{ID: newBeatID1, Title: "Le débat", Color: "#e0457b", Description: "Marie hésite.", Prompt: "Lent."},
					{ID: newBeatID2, Title: "Le passage", Color: "#2bb673"},
				},
				Now: now,
			},

			expect: []*dao.Beat{
				{
					ID: newBeatID1, ProjectID: projectID, Position: 3, Title: "Le débat", Color: "#e0457b",
					Description: "Marie hésite.", Prompt: "Lent.", UpdatedAt: now,
				},
				{ID: newBeatID2, ProjectID: projectID, Position: 4, Title: "Le passage", Color: "#2bb673", UpdatedAt: now},
			},
		},
		{
			name: "Success/PastTheEnd",

			request: &dao.BeatsInsertRequest{
				ProjectID: projectID,
				Position:  199,
				Beats:     []dao.BeatsInsertItem{{ID: newBeatID1, Title: "Le débat", Color: "#e0457b"}},
				Now:       now,
			},

			expect: []*dao.Beat{
				{ID: newBeatID1, ProjectID: projectID, Position: 3, Title: "Le débat", Color: "#e0457b", UpdatedAt: now},
			},
		},
		{
			name: "Error/ProjectFull",

			request: &dao.BeatsInsertRequest{
				ProjectID: projectID,
				Position:  0,
				Beats:     []dao.BeatsInsertItem{{ID: newBeatID1, Title: "Le débat", Color: "#e0457b"}},
				Now:       now,
			},

			full:      true,
			expectErr: dao.ErrBeatsInsertFull,
		},
		{
			name: "Success/Shift",

			request: &dao.BeatsInsertRequest{
				ProjectID: projectID,
				Position:  1,
				Beats: []dao.BeatsInsertItem{
					{ID: newBeatID1, Title: "Le débat", Color: "#e0457b"},
					{ID: newBeatID2, Title: "Le passage", Color: "#2bb673"},
				},
				Now: now,
			},

			expect: []*dao.Beat{
				{ID: newBeatID1, ProjectID: projectID, Position: 1, Title: "Le débat", Color: "#e0457b", UpdatedAt: now},
				{ID: newBeatID2, ProjectID: projectID, Position: 2, Title: "Le passage", Color: "#2bb673", UpdatedAt: now},
			},
			expectBeats: []*dao.Beat{
				composerFixtures().beats[0],
				{ID: newBeatID1, ProjectID: projectID, Position: 1, Title: "Le débat", Color: "#e0457b", UpdatedAt: now},
				{ID: newBeatID2, ProjectID: projectID, Position: 2, Title: "Le passage", Color: "#2bb673", UpdatedAt: now},
				beatAt(composerFixtures().beats[1], 3),
				beatAt(composerFixtures().beats[2], 4),
			},
		},
	}

	daoBeatsInsert := dao.NewPgBeatsInsert()

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

				result, err := daoBeatsInsert.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)

				slices.SortFunc(result, func(a, b *dao.Beat) int { return cmp.Compare(a.Position, b.Position) })
				require.Equal(t, testCase.expect, result)

				if testCase.expectBeats != nil {
					require.Equal(t, testCase.expectBeats, listBeats(ctx, t, projectID))
				}
			})
		})
	}
}
