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

func TestPgProjectInsert(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *dao.ProjectInsertRequest

		expect    *dao.Project
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.ProjectInsertRequest{
				ID:       projectID,
				OwnerID:  ownerID,
				Title:    "Les Heures claires",
				Premise:  "Marie attend.",
				Language: "fr",
				Now:      now,
			},

			expect: &dao.Project{
				ID:        projectID,
				OwnerID:   ownerID,
				Title:     "Les Heures claires",
				Premise:   "Marie attend.",
				Language:  "fr",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	daoProjectInsert := dao.NewPgProjectInsert()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				result, err := daoProjectInsert.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
