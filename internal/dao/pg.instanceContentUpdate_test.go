package dao_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/v2/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
	"github.com/a-novel/service-narrative-engine/v2/internal/models/migrations"
)

func TestPgInstanceContentUpdate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string

		request *dao.InstanceContentUpdateRequest

		// expect is the second fixture instance after the write.
		expect    *dao.Instance
		expectErr error
	}{
		{
			name: "Success",

			request: &dao.InstanceContentUpdateRequest{
				ProjectID: projectID, ID: otherInstanceID, Version: "0.2.0", Content: json.RawMessage(instanceContent),
				UpdatedAt: documentUpdatedAt, Now: now,
			},

			expect: &dao.Instance{
				ID: otherInstanceID, ProjectID: projectID, Plugin: "characters", Version: "0.2.0",
				Name: "Characters", CreatedAt: createdAt, UpdatedAt: now,
				Content: json.RawMessage(instanceContent),
			},
		},
		{
			name: "Error/Stale",

			request: &dao.InstanceContentUpdateRequest{
				ProjectID: projectID, ID: otherInstanceID, Version: "0.2.0", Content: json.RawMessage(instanceContent),
				UpdatedAt: updatedAt, Now: now,
			},

			expect:    instanceFixtures().instances[1],
			expectErr: dao.ErrInstanceContentUpdateStale,
		},
		{
			// The instant compares to the microsecond: the value a JavaScript date keeps names another write.
			name: "Error/Stale/Millisecond",

			request: &dao.InstanceContentUpdateRequest{
				ProjectID: projectID, ID: otherInstanceID, Version: "0.2.0", Content: json.RawMessage(instanceContent),
				UpdatedAt: documentUpdatedAt.Truncate(time.Millisecond), Now: now,
			},

			expect:    instanceFixtures().instances[1],
			expectErr: dao.ErrInstanceContentUpdateStale,
		},
		{
			name: "Error/NotFound",

			request: &dao.InstanceContentUpdateRequest{
				ProjectID: projectID, ID: absentID, Version: "0.2.0", Content: json.RawMessage(instanceContent),
				UpdatedAt: documentUpdatedAt, Now: now,
			},

			expect:    instanceFixtures().instances[1],
			expectErr: dao.ErrInstanceNotFound,
		},
		{
			name: "Error/OtherProject",

			request: &dao.InstanceContentUpdateRequest{
				ProjectID: otherProjectID, ID: otherInstanceID, Version: "0.2.0",
				Content: json.RawMessage(instanceContent), UpdatedAt: documentUpdatedAt, Now: now,
			},

			expect:    instanceFixtures().instances[1],
			expectErr: dao.ErrInstanceNotFound,
		},
	}

	daoInstanceContentUpdate := dao.NewPgInstanceContentUpdate()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, instanceFixtures())

				result, err := daoInstanceContentUpdate.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)

				if testCase.expectErr == nil {
					require.Equal(t, &testCase.expect.InstanceDescriptor, result)
				} else {
					require.Nil(t, result)
				}

				// The write replaces the instance it names and leaves the other one of the project.
				instances := projectInstances(ctx, t, projectID)
				require.Equal(t, []*dao.Instance{instanceFixtures().instances[0], testCase.expect}, instances)

				// A content write leaves the links of every instance, and the stamp a links write names.
				require.Equal(t, instanceFixtures().allLinks(), storedInstanceLinks(ctx, t, projectID))
				requireStoredBytes(ctx, t)
			})
		})
	}
}
