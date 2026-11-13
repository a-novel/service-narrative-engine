package dao_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgProjectEditsList(t *testing.T) {
	t.Parallel()

	// precise keeps the microseconds the database stores.
	precise := time.Date(2021, 1, 3, 8, 15, 30, 123456000, time.UTC)

	project := func(id string, owner uuid.UUID, editedAt time.Time) *dao.Project {
		return &dao.Project{
			ID: uuid.MustParse(id), OwnerID: owner, Title: "Project " + id[len(id)-1:], Language: "en", Tags: []string{},
			Genres: []string{}, CreatedAt: createdAt, UpdatedAt: updatedAt, EditedAt: editedAt,
		}
	}

	first := project("00000000-0000-0000-0000-000000000001", ownerID, precise)
	second := project("00000000-0000-0000-0000-000000000003", ownerID, updatedAt)
	foreign := project("00000000-0000-0000-0000-000000000002", otherOwnerID, now)

	testCases := []struct {
		name string

		request *dao.ProjectEditsListRequest

		expect []*dao.ProjectEditsListRow
	}{
		{
			name: "Success",

			request: &dao.ProjectEditsListRequest{OwnerID: ownerID, Limit: 10},

			expect: []*dao.ProjectEditsListRow{
				{ID: first.ID, EditedAt: first.EditedAt},
				{ID: second.ID, EditedAt: second.EditedAt},
			},
		},
		{
			name: "Success/Limit",

			request: &dao.ProjectEditsListRequest{OwnerID: ownerID, Limit: 0},
		},
		{
			name: "Success/NoProject",

			request: &dao.ProjectEditsListRequest{OwnerID: uuid.MustParse("00000000-0000-0000-0000-0000000000a3"), Limit: 10},
		},
	}

	daoProjectEditsList := dao.NewPgProjectEditsList()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, fixtures{projects: []*dao.Project{first, foreign, second}})

				result, err := daoProjectEditsList.Exec(ctx, testCase.request)
				require.NoError(t, err)
				// The rows come in no order.
				require.ElementsMatch(t, testCase.expect, result)
			})
		})
	}
}
