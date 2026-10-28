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

	row := func(project *dao.Project) *dao.ProjectListRow {
		return &dao.ProjectListRow{Project: *project, Total: 3}
	}

	testCases := []struct {
		name string

		request *dao.ProjectListRequest

		expect    []*dao.ProjectListRow
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.ProjectListRequest{OwnerID: ownerID, Limit: 10},

			expect: []*dao.ProjectListRow{row(third), row(second), row(first)},
		},
		{
			name: "Success/Page",

			request: &dao.ProjectListRequest{OwnerID: ownerID, Limit: 1, Offset: 1},

			expect: []*dao.ProjectListRow{row(second)},
		},
		{
			name: "Success/NoProject",

			request: &dao.ProjectListRequest{OwnerID: uuid.MustParse("00000000-0000-0000-0000-0000000000a3"), Limit: 10},
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
