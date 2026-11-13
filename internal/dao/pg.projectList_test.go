package dao_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgProjectList(t *testing.T) {
	t.Parallel()

	project := func(id string, owner uuid.UUID, editedAt time.Time) *dao.Project {
		return &dao.Project{
			ID: uuid.MustParse(id), OwnerID: owner, Title: "Project " + id[len(id)-1:], Language: "en", Tags: []string{},
			Genres: []string{}, CreatedAt: createdAt, UpdatedAt: updatedAt, EditedAt: editedAt,
		}
	}

	// The oldest project holds the latest edit, and the two others share theirs, so the newer of them
	// comes first.
	first := project("00000000-0000-0000-0000-000000000001", ownerID, now)
	second := project("00000000-0000-0000-0000-000000000003", ownerID, updatedAt)
	third := project("00000000-0000-0000-0000-000000000004", ownerID, updatedAt)
	foreign := project("00000000-0000-0000-0000-000000000002", otherOwnerID, now)

	// The first and the third project hold logos, at distinct versions, and the second none.
	first.LogoUpdatedAt = lo.ToPtr(now)
	first.LogoChecksum = logoChecksum([]byte("logo"))
	third.LogoUpdatedAt = lo.ToPtr(createdAt)
	third.LogoChecksum = logoChecksum([]byte("logo"))

	logo := func(project *dao.Project, version time.Time) *dao.ProjectLogo {
		return &dao.ProjectLogo{ProjectID: project.ID, Data: []byte("logo"), MediaType: "image/webp", UpdatedAt: version}
	}
	logos := []*dao.ProjectLogo{logo(first, now), logo(third, createdAt)}

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

			expect: []*dao.ProjectListRow{row(first), row(third), row(second)},
		},
		{
			name: "Success/Page",

			request: &dao.ProjectListRequest{OwnerID: ownerID, Limit: 1, Offset: 1},

			expect: []*dao.ProjectListRow{row(third)},
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

				insertFixtures(ctx, t, fixtures{projects: []*dao.Project{first, foreign, second, third}, logos: logos})

				result, err := daoProjectList.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
