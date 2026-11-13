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

func TestPgProjectFieldsDrift(t *testing.T) {
	t.Parallel()

	// precise keeps the microseconds the database stores, which a served updatedAt carries.
	precise := time.Date(2021, 1, 2, 8, 15, 30, 123456000, time.UTC)

	project := func(id string, owner uuid.UUID, updated time.Time) *dao.Project {
		return &dao.Project{
			ID: uuid.MustParse(id), OwnerID: owner, Title: "Project " + id[len(id)-1:], Language: "en", Tags: []string{},
			Genres: []string{}, CreatedAt: createdAt, UpdatedAt: updated, EditedAt: now,
		}
	}

	first := project("00000000-0000-0000-0000-000000000001", ownerID, precise)
	second := project("00000000-0000-0000-0000-000000000003", ownerID, updatedAt)
	third := project("00000000-0000-0000-0000-000000000004", ownerID, updatedAt)
	foreign := project("00000000-0000-0000-0000-000000000002", otherOwnerID, updatedAt)

	// served writes an instant as the JSON of a project carries it.
	served := func(instant time.Time) string {
		return instant.Format(time.RFC3339Nano)
	}

	// The first project holds a logo, which a drift leaves out.
	logos := []*dao.ProjectLogo{
		{ProjectID: first.ID, Data: []byte("logo"), MediaType: "image/webp", UpdatedAt: now},
	}

	drifted := func(project *dao.Project) *dao.ProjectFieldsDriftRow {
		return &dao.ProjectFieldsDriftRow{Project: *project}
	}

	removed := func(id uuid.UUID) *dao.ProjectFieldsDriftRow {
		return &dao.ProjectFieldsDriftRow{RemovedID: &id}
	}

	testCases := []struct {
		name string

		request *dao.ProjectFieldsDriftRequest

		expect []*dao.ProjectFieldsDriftRow
	}{
		{
			name: "Success/NothingHeld",

			request: &dao.ProjectFieldsDriftRequest{OwnerID: ownerID, Projects: map[string]string{}, Limit: 10},

			expect: []*dao.ProjectFieldsDriftRow{drifted(first), drifted(second), drifted(third)},
		},
		{
			name: "Success/InSync",

			request: &dao.ProjectFieldsDriftRequest{
				OwnerID: ownerID,
				Projects: map[string]string{
					first.ID.String():  served(first.UpdatedAt),
					second.ID.String(): served(second.UpdatedAt),
					third.ID.String():  served(third.UpdatedAt),
				},
				Limit: 10,
			},
		},
		{
			// The held instant matches the stored one in another offset.
			name: "Success/InSyncOffset",

			request: &dao.ProjectFieldsDriftRequest{
				OwnerID: ownerID,
				Projects: map[string]string{
					first.ID.String():  first.UpdatedAt.In(time.FixedZone("", 2*60*60)).Format(time.RFC3339Nano),
					second.ID.String(): served(second.UpdatedAt),
					third.ID.String():  served(third.UpdatedAt),
				},
				Limit: 10,
			},
		},
		{
			// A JavaScript date keeps the milliseconds of the stored instant, and holds it in sync.
			name: "Success/InSyncMilliseconds",

			request: &dao.ProjectFieldsDriftRequest{
				OwnerID: ownerID,
				Projects: map[string]string{
					first.ID.String():  served(first.UpdatedAt.Truncate(time.Millisecond)),
					second.ID.String(): served(second.UpdatedAt),
					third.ID.String():  served(third.UpdatedAt),
				},
				Limit: 10,
			},
		},
		{
			// The first project is held a millisecond early, the third not at all, and the foreign and the absent
			// IDs name no project of the owner.
			name: "Success/Drift",

			request: &dao.ProjectFieldsDriftRequest{
				OwnerID: ownerID,
				Projects: map[string]string{
					first.ID.String():   served(first.UpdatedAt.Truncate(time.Millisecond).Add(-time.Millisecond)),
					second.ID.String():  served(second.UpdatedAt),
					foreign.ID.String(): served(foreign.UpdatedAt),
					absentID.String():   served(updatedAt),
				},
				Limit: 10,
			},

			expect: []*dao.ProjectFieldsDriftRow{
				drifted(first), drifted(third), removed(foreign.ID), removed(absentID),
			},
		},
		{
			name: "Success/Limit",

			request: &dao.ProjectFieldsDriftRequest{OwnerID: ownerID, Projects: map[string]string{}, Limit: 0},
		},
		{
			name: "Success/NoProject",

			request: &dao.ProjectFieldsDriftRequest{
				OwnerID:  uuid.MustParse("00000000-0000-0000-0000-0000000000a3"),
				Projects: map[string]string{first.ID.String(): served(first.UpdatedAt)},
				Limit:    10,
			},

			expect: []*dao.ProjectFieldsDriftRow{removed(first.ID)},
		},
	}

	daoProjectFieldsDrift := dao.NewPgProjectFieldsDrift()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, fixtures{projects: []*dao.Project{first, foreign, second, third}, logos: logos})

				result, err := daoProjectFieldsDrift.Exec(ctx, testCase.request)
				require.NoError(t, err)
				// A full join returns its rows in no order.
				require.ElementsMatch(t, testCase.expect, result)
			})
		})
	}
}
