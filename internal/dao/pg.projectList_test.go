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

func TestPgProjectList(t *testing.T) {
	t.Parallel()

	project := func(id string, owner uuid.UUID) *dao.Project {
		return &dao.Project{
			ID: uuid.MustParse(id), OwnerID: owner, Title: "Project " + id[len(id)-1:], Language: "en",
			CreatedAt: createdAt, UpdatedAt: updatedAt,
		}
	}

	first := project("00000000-0000-0000-0000-000000000001", ownerID)
	second := project("00000000-0000-0000-0000-000000000003", ownerID)
	third := project("00000000-0000-0000-0000-000000000004", ownerID)
	foreign := project("00000000-0000-0000-0000-000000000002", otherOwnerID)

	testCases := []struct {
		name string

		request *dao.ProjectListRequest

		expect    []*dao.Project
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.ProjectListRequest{OwnerID: ownerID, Limit: 10, After: uuid.Max},

			expect: []*dao.Project{third, second, first},
		},
		{
			name: "Success/After",

			request: &dao.ProjectListRequest{OwnerID: ownerID, Limit: 10, After: third.ID},

			expect: []*dao.Project{second, first},
		},
		{
			name: "Success/Limit",

			request: &dao.ProjectListRequest{OwnerID: ownerID, Limit: 1, After: uuid.Max},

			expect: []*dao.Project{third},
		},
		{
			name: "Success/NoProject",

			request: &dao.ProjectListRequest{
				OwnerID: uuid.MustParse("00000000-0000-0000-0000-0000000000a3"), Limit: 10, After: uuid.Max,
			},
		},
	}

	daoProjectList := dao.NewPgProjectList()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, fixtures{projects: []*dao.Project{first, foreign, second, third}})

				result, err := daoProjectList.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
