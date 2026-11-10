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

func TestPgProjectCount(t *testing.T) {
	t.Parallel()

	// secondProject gives the owner two projects beside the one of the other owner.
	secondProject := &dao.Project{
		ID: uuid.MustParse("00000000-0000-0000-0000-000000000003"), OwnerID: ownerID, Title: "Second",
		Language: "en", Tags: []string{}, Genres: []string{}, CreatedAt: createdAt, UpdatedAt: updatedAt,
	}

	testCases := []struct {
		name string

		request *dao.ProjectCountRequest

		expect *dao.ProjectCount
	}{
		{
			name: "Success",

			request: &dao.ProjectCountRequest{OwnerID: ownerID},

			expect: &dao.ProjectCount{Count: 2},
		},
		{
			name: "Success/OtherOwner",

			request: &dao.ProjectCountRequest{OwnerID: otherOwnerID},

			expect: &dao.ProjectCount{Count: 1},
		},
		{
			name: "Success/None",

			request: &dao.ProjectCountRequest{OwnerID: absentID},

			expect: &dao.ProjectCount{Count: 0},
		},
	}

	daoProjectCount := dao.NewPgProjectCount()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				data := composerFixtures()
				data.projects = append(data.projects, secondProject)
				insertFixtures(ctx, t, data)

				result, err := daoProjectCount.Exec(ctx, testCase.request)
				require.NoError(t, err)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
