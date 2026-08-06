package dao_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgProjectSelect(t *testing.T) {
	t.Parallel()

	otherOwnerID := uuid.MustParse("00000000-0000-0000-0000-000000000043")
	absentProjectID := uuid.MustParse("00000000-0000-0000-0000-000000000399")
	testCases := []struct {
		name string

		request *dao.ProjectSelectRequest

		expect    *dao.Project
		expectErr error
	}{
		{
			name: "Success",
			request: &dao.ProjectSelectRequest{
				ID: fixtureProjectID, OwnerID: fixtureOwnerID,
			},
			expect: &dao.Project{
				ID: fixtureProjectID, OwnerID: fixtureOwnerID, CreatedAt: fixtureCreatedAt,
			},
		},
		{
			name: "Error/OtherOwner",
			request: &dao.ProjectSelectRequest{
				ID: fixtureProjectID, OwnerID: otherOwnerID,
			},
			expectErr: dao.ErrProjectSelectNotFound,
		},
		{
			name: "Error/Absent",
			request: &dao.ProjectSelectRequest{
				ID: absentProjectID, OwnerID: fixtureOwnerID,
			},
			expectErr: dao.ErrProjectSelectNotFound,
		},
	}

	operation := dao.NewPgProjectSelect()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunIsolatedTransactionalTest(
				t,
				configtest.PostgresPreset,
				migrations.Migrations,
				func(ctx context.Context, t *testing.T) {
					t.Helper()

					insertWalkingSkeletonFixtures(t, ctx)

					project, err := operation.Exec(ctx, testCase.request)
					require.ErrorIs(t, err, testCase.expectErr)
					require.Equal(t, testCase.expect, project)
				},
			)
		})
	}
}
