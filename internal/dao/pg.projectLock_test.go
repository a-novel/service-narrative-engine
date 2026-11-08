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

func TestPgProjectLock(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *dao.ProjectLockRequest

		expect bool
		// expectEditedAt is the edited_at of each fixture project after the lock.
		expectEditedAt map[uuid.UUID]time.Time
	}{
		{
			name: "Success",

			request: &dao.ProjectLockRequest{ID: projectID, OwnerID: ownerID, Now: now},

			expect:         true,
			expectEditedAt: map[uuid.UUID]time.Time{projectID: now, otherProjectID: updatedAt},
		},
		{
			name: "Success/OtherOwner",

			request: &dao.ProjectLockRequest{ID: otherProjectID, OwnerID: ownerID, Now: now},

			expect:         false,
			expectEditedAt: map[uuid.UUID]time.Time{projectID: updatedAt, otherProjectID: updatedAt},
		},
		{
			name: "Success/NotFound",

			request: &dao.ProjectLockRequest{ID: absentID, OwnerID: ownerID, Now: now},

			expect:         false,
			expectEditedAt: map[uuid.UUID]time.Time{projectID: updatedAt, otherProjectID: updatedAt},
		},
	}

	daoProjectLock := dao.NewPgProjectLock()
	daoProjectSelect := dao.NewPgProjectSelect()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoProjectLock.Exec(ctx, testCase.request)
				require.NoError(t, err)
				require.Equal(t, testCase.expect, result)

				editedAt := map[uuid.UUID]time.Time{}

				for id := range testCase.expectEditedAt {
					project, err := daoProjectSelect.Exec(ctx, &dao.ProjectSelectRequest{ID: id})
					require.NoError(t, err)

					editedAt[id] = project.EditedAt
				}

				require.Equal(t, testCase.expectEditedAt, editedAt)
			})
		})
	}
}
