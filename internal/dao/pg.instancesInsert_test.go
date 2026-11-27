package dao_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/v2/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/v2/internal/dao"
	"github.com/a-novel/service-narrative-engine/v2/internal/models/migrations"
)

func TestPgInstancesInsert(t *testing.T) {
	t.Parallel()

	firstID := uuid.MustParse("00000000-0000-0000-0000-000000000051")
	secondID := uuid.MustParse("00000000-0000-0000-0000-000000000052")

	items := []dao.InstancesInsertItem{
		{
			ID: firstID, Plugin: "scenes-composer", Version: "1.2.0", Name: "Draft",
			Content: json.RawMessage(instanceContentNext),
		},
		{ID: secondID, Plugin: "characters", Version: "0.1.0", Name: "Cast", Content: json.RawMessage(`[]`)},
	}

	inserted := []*dao.InstanceDescriptor{
		{
			ID: firstID, ProjectID: projectID, Plugin: "scenes-composer", Version: "1.2.0", Name: "Draft",
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: secondID, ProjectID: projectID, Plugin: "characters", Version: "0.1.0", Name: "Cast",
			CreatedAt: now, UpdatedAt: now,
		},
	}

	testCases := []struct {
		name string

		fixtures fixtures
		request  *dao.InstancesInsertRequest

		expect []*dao.InstanceDescriptor
		// expectCount is how many instances the test project holds after the insert.
		expectCount int
		// expectDefault is the default instance of the test project after the insert.
		expectDefault *uuid.UUID
	}{
		{
			// The insert leaves the default of the project to its caller.
			name: "Success",

			fixtures: projectFixtures(),
			request:  &dao.InstancesInsertRequest{ProjectID: projectID, Instances: items, Now: now},

			expect:      inserted,
			expectCount: 2,
		},
		{
			name: "Success/KeepsDefault",

			fixtures: instanceFixtures(),
			request:  &dao.InstancesInsertRequest{ProjectID: projectID, Instances: items, Now: now},

			expect:        inserted,
			expectCount:   4,
			expectDefault: new(instanceID),
		},
		{
			// The rows come back in the order of the request, whatever order their IDs sort in.
			name: "Success/RequestOrder",

			fixtures: projectFixtures(),
			request: &dao.InstancesInsertRequest{
				ProjectID: projectID, Instances: []dao.InstancesInsertItem{items[1], items[0]}, Now: now,
			},

			expect:      []*dao.InstanceDescriptor{inserted[1], inserted[0]},
			expectCount: 2,
		},
	}

	daoInstancesInsert := dao.NewPgInstancesInsert()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				insertFixtures(ctx, t, testCase.fixtures)

				result, err := daoInstancesInsert.Exec(ctx, testCase.request)
				require.NoError(t, err)
				require.Equal(t, testCase.expect, result)

				require.Len(t, projectInstances(ctx, t, projectID), testCase.expectCount)
				require.Equal(t, testCase.expectDefault, projectDefault(ctx, t, projectID))
				requireStoredBytes(ctx, t)
			})
		})
	}
}
