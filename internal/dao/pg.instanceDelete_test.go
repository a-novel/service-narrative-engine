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

func TestPgInstanceDelete(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *dao.InstanceDeleteRequest

		expect *dao.InstanceDescriptor
		// expectInstances are the instances of the test project after the delete.
		expectInstances []*dao.Instance
		expectErr       error
	}{
		{
			name: "Success",

			request: &dao.InstanceDeleteRequest{ProjectID: projectID, ID: otherInstanceID},

			expect:          instanceFixtures().descriptors(otherInstanceID)[0],
			expectInstances: instanceFixtures().instances[:1],
		},
		{
			// The test project opens on its first instance.
			name: "Error/Default",

			request: &dao.InstanceDeleteRequest{ProjectID: projectID, ID: instanceID},

			expectInstances: instanceFixtures().instances[:2],
			expectErr:       dao.ErrInstanceDeleteDefault,
		},
		{
			name: "Error/NotFound",

			request: &dao.InstanceDeleteRequest{ProjectID: projectID, ID: absentID},

			expectInstances: instanceFixtures().instances[:2],
			expectErr:       dao.ErrInstanceNotFound,
		},
		{
			name: "Error/OtherProject",

			request: &dao.InstanceDeleteRequest{ProjectID: otherProjectID, ID: otherInstanceID},

			expectInstances: instanceFixtures().instances[:2],
			expectErr:       dao.ErrInstanceNotFound,
		},
	}

	daoInstanceDelete := dao.NewPgInstanceDelete()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, instanceFixtures())

				result, err := daoInstanceDelete.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)

				if testCase.expectErr == nil {
					require.Equal(t, testCase.expect, result)
				} else {
					require.Nil(t, result)
				}

				require.Equal(t, testCase.expectInstances, projectInstances(ctx, t, projectID))
				requireStoredBytes(ctx, t)
			})
		})
	}
}
