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

func TestPgProjectUpdate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *dao.ProjectUpdateRequest

		expect    *dao.Project
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.ProjectUpdateRequest{
				ID: projectID, OwnerID: ownerID, Revision: 1,
				Title: "Clear Hours", Premise: "Marie waits.", Language: "en", Now: now,
			},

			expect: &dao.Project{
				ID: projectID, OwnerID: ownerID, Title: "Clear Hours", Premise: "Marie waits.", Language: "en",
				Revision: 2, CreatedAt: createdAt, UpdatedAt: now,
			},
		},
		{
			name: "Error/StaleRevision",

			request: &dao.ProjectUpdateRequest{
				ID: projectID, OwnerID: ownerID, Revision: 2, Title: "Clear Hours", Language: "en", Now: now,
			},

			expectErr: dao.ErrProjectUpdateNotFound,
		},
		{
			name: "Error/OtherOwner",

			request: &dao.ProjectUpdateRequest{
				ID: projectID, OwnerID: otherOwnerID, Revision: 1, Title: "Clear Hours", Language: "en", Now: now,
			},

			expectErr: dao.ErrProjectUpdateNotFound,
		},
	}

	daoProjectUpdate := dao.NewPgProjectUpdate()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, composerFixtures())

				result, err := daoProjectUpdate.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)
			})
		})
	}
}
