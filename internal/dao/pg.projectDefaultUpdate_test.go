package dao_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/v2/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
	"github.com/a-novel/service-narrative-engine/v2/internal/models/migrations"
)

func TestPgProjectDefaultUpdate(t *testing.T) {
	t.Parallel()

	// chosen returns the test project once it opens on its second instance. The choice stamps its fields.
	chosen := func() *dao.Project {
		project := instanceFixtures().projects[0]
		project.DefaultInstanceID = new(otherInstanceID)
		project.UpdatedAt = now

		return project
	}

	testCases := []struct {
		name string

		request *dao.ProjectDefaultUpdateRequest

		expect *dao.Project
		// expectDefault is the default instance of the test project after the write.
		expectDefault *uuid.UUID
		expectErr     error
	}{
		{
			name: "Success",

			request: &dao.ProjectDefaultUpdateRequest{ProjectID: projectID, InstanceID: otherInstanceID, Now: now},

			expect:        chosen(),
			expectDefault: new(otherInstanceID),
		},
		{
			name: "Error/InstanceNotFound",

			request: &dao.ProjectDefaultUpdateRequest{ProjectID: projectID, InstanceID: absentID, Now: now},

			expectDefault: new(instanceID),
			expectErr:     dao.ErrInstanceNotFound,
		},
		{
			// The instance exists, in the project of another owner.
			name: "Error/InstanceOfOtherProject",

			request: &dao.ProjectDefaultUpdateRequest{ProjectID: projectID, InstanceID: alienInstanceID, Now: now},

			expectDefault: new(instanceID),
			expectErr:     dao.ErrInstanceNotFound,
		},
		{
			name: "Error/NotFound",

			request: &dao.ProjectDefaultUpdateRequest{ProjectID: absentID, InstanceID: instanceID, Now: now},

			expectDefault: new(instanceID),
			expectErr:     dao.ErrProjectNotFound,
		},
	}

	daoProjectDefaultUpdate := dao.NewPgProjectDefaultUpdate()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, instanceFixtures())

				result, err := daoProjectDefaultUpdate.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)
				require.Equal(t, testCase.expect, result)

				require.Equal(t, testCase.expectDefault, projectDefault(ctx, t, projectID))
			})
		})
	}
}
