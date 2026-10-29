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

func TestPgBeatUpdate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *dao.BeatUpdateRequest

		expect    *dao.Beat
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.BeatUpdateRequest{
				ProjectID: projectID, ID: beatID2, Title: "Le mensonge", Color: "#e0457b", Description: "Paul ment.",
				Prompt: "Garder le suspense.", Now: now,
			},

			expect: &dao.Beat{
				ID: beatID2, ProjectID: projectID, Position: 1, Title: "Le mensonge", Color: "#e0457b",
				Description: "Paul ment.", Prompt: "Garder le suspense.", UpdatedAt: now,
			},
		},
		{
			name: "Error/NotFound",

			request: &dao.BeatUpdateRequest{
				ProjectID: projectID, ID: absentID, Title: "Le mensonge", Color: "#e0457b", Now: now,
			},

			expectErr: dao.ErrBeatUpdateNotFound,
		},
		{
			name: "Error/OtherProject",

			request: &dao.BeatUpdateRequest{
				ProjectID: projectID, ID: otherBeatID, Title: "Le mensonge", Color: "#e0457b", Now: now,
			},

			expectErr: dao.ErrBeatUpdateNotFound,
		},
	}

	daoBeatUpdate := dao.NewPgBeatUpdate()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoBeatUpdate.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
