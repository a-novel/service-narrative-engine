package dao_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/v2/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
	"github.com/a-novel/service-narrative-engine/v2/internal/models/migrations"
)

func TestPgInstanceUpdate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *dao.InstanceUpdateRequest

		expect    *dao.InstanceDescriptor
		expectErr error
	}{
		{
			// The rename leaves the instance's updated_at, which its content writes name.
			name: "Success",

			request: &dao.InstanceUpdateRequest{ProjectID: projectID, ID: instanceID, Name: "Outline"},

			expect: &dao.InstanceDescriptor{
				ID: instanceID, ProjectID: projectID, Plugin: "scenes-composer", Version: "1.0.0", Name: "Outline",
				CreatedAt: createdAt, UpdatedAt: documentUpdatedAt,
			},
		},
		{
			name: "Error/NotFound",

			request: &dao.InstanceUpdateRequest{ProjectID: projectID, ID: absentID, Name: "Outline"},

			expectErr: dao.ErrInstanceNotFound,
		},
		{
			name: "Error/OtherProject",

			request: &dao.InstanceUpdateRequest{ProjectID: projectID, ID: alienInstanceID, Name: "Outline"},

			expectErr: dao.ErrInstanceNotFound,
		},
	}

	daoInstanceUpdate := dao.NewPgInstanceUpdate()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, instanceFixtures())

				result, err := daoInstanceUpdate.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)

				// A refused rename leaves every instance as it was.
				if testCase.expectErr != nil {
					require.Equal(t, instanceFixtures().instances[:2], projectInstances(ctx, t, projectID))
					require.Equal(t, instanceFixtures().instances[2:], projectInstances(ctx, t, otherProjectID))
				}

				requireStoredBytes(ctx, t)
			})
		})
	}
}
