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

func TestPgProjectLogosDrift(t *testing.T) {
	t.Parallel()

	project := func(id string, owner uuid.UUID) *dao.Project {
		return &dao.Project{
			ID: uuid.MustParse(id), OwnerID: owner, Title: "Project " + id[len(id)-1:], Language: "en", Tags: []string{},
			Genres: []string{}, CreatedAt: createdAt, UpdatedAt: updatedAt, EditedAt: updatedAt,
		}
	}

	// The first and the second project hold logos, the third none, and the foreign one belongs to another owner.
	first := project("00000000-0000-0000-0000-000000000001", ownerID)
	second := project("00000000-0000-0000-0000-000000000003", ownerID)
	third := project("00000000-0000-0000-0000-000000000004", ownerID)
	foreign := project("00000000-0000-0000-0000-000000000002", otherOwnerID)

	logo := func(project *dao.Project, data string) *dao.ProjectLogo {
		return &dao.ProjectLogo{ProjectID: project.ID, Data: []byte(data), MediaType: "image/webp", UpdatedAt: now}
	}
	logos := []*dao.ProjectLogo{logo(first, "first logo"), logo(second, "second logo"), logo(foreign, "foreign logo")}

	firstChecksum := logoChecksum(logos[0].Data)
	secondChecksum := logoChecksum(logos[1].Data)
	foreignChecksum := logoChecksum(logos[2].Data)

	testCases := []struct {
		name string

		request *dao.ProjectLogosDriftRequest

		expect []*dao.ProjectLogosDriftRow
	}{
		{
			name: "Success/NothingHeld",

			request: &dao.ProjectLogosDriftRequest{OwnerID: ownerID, Logos: map[string]string{}, Limit: 10},

			expect: []*dao.ProjectLogosDriftRow{
				{ProjectID: first.ID, Checksum: firstChecksum},
				{ProjectID: second.ID, Checksum: secondChecksum},
			},
		},
		{
			name: "Success/InSync",

			request: &dao.ProjectLogosDriftRequest{
				OwnerID: ownerID,
				Logos:   map[string]string{first.ID.String(): *firstChecksum, second.ID.String(): *secondChecksum},
				Limit:   10,
			},
		},
		{
			// The first logo is held with another checksum, the second not at all. The third project holds no logo,
			// the foreign one is another owner's, and the absent ID names no project.
			name: "Success/Drift",

			request: &dao.ProjectLogosDriftRequest{
				OwnerID: ownerID,
				Logos: map[string]string{
					first.ID.String():   *secondChecksum,
					third.ID.String():   *firstChecksum,
					foreign.ID.String(): *foreignChecksum,
					absentID.String():   *firstChecksum,
				},
				Limit: 10,
			},

			expect: []*dao.ProjectLogosDriftRow{
				{ProjectID: first.ID, Checksum: firstChecksum},
				{ProjectID: second.ID, Checksum: secondChecksum},
				{ProjectID: third.ID},
				{ProjectID: foreign.ID},
				{ProjectID: absentID},
			},
		},
		{
			name: "Success/Limit",

			request: &dao.ProjectLogosDriftRequest{OwnerID: ownerID, Logos: map[string]string{}, Limit: 0},
		},
	}

	daoProjectLogosDrift := dao.NewPgProjectLogosDrift()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, fixtures{projects: []*dao.Project{first, foreign, second, third}, logos: logos})

				result, err := daoProjectLogosDrift.Exec(ctx, testCase.request)
				require.NoError(t, err)
				// A full join returns its rows in no order.
				require.ElementsMatch(t, testCase.expect, result)
			})
		})
	}
}
