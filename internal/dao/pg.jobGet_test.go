package dao_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestJobGet(t *testing.T) {
	t.Parallel()

	ownerAlice := uuid.MustParse("00000000-0000-0000-0000-00000000a11c")
	ownerBob := uuid.MustParse("00000000-0000-0000-0000-00000000b0b0")

	jobID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Seeding through the enqueue DAO rather than a hand-built row keeps the expectation exact: every
	// column the database fills in is whatever it actually filled in.
	aliceJob := &dao.JobEnqueueRequest{
		ID:                 jobID,
		Kind:               "generate",
		Payload:            json.RawMessage(`{"seed":"an idea"}`),
		OwnerID:            ownerAlice,
		RequestFingerprint: []byte{0x01, 0x02},
		MaxAttempts:        1,
	}

	testCases := []struct {
		name string

		fixtures []*dao.JobEnqueueRequest

		request *dao.JobGetRequest

		// expectFixture indexes fixtures when a row is expected back; -1 expects an error instead.
		expectFixture int
		expectErr     error
	}{
		{
			name: "Success",

			fixtures: []*dao.JobEnqueueRequest{aliceJob},

			request: &dao.JobGetRequest{ID: jobID, OwnerID: ownerAlice},

			expectFixture: 0,
		},
		{
			name: "Error/NotFound",

			request: &dao.JobGetRequest{ID: jobID, OwnerID: ownerAlice},

			expectFixture: -1,
			expectErr:     dao.ErrJobGetNotFound,
		},
		{
			// The ownership regression for the whole epic. Reading another owner's job by its exact
			// id reports not-found, so the response cannot confirm the job exists. A handler mapping
			// this to 403 would hand back the existence oracle the predicate exists to remove.
			name: "Error/OtherOwner",

			fixtures: []*dao.JobEnqueueRequest{aliceJob},

			request: &dao.JobGetRequest{ID: jobID, OwnerID: ownerBob},

			expectFixture: -1,
			expectErr:     dao.ErrJobGetNotFound,
		},
	}

	daoJobEnqueue := dao.NewJobEnqueue()
	daoJobGet := dao.NewJobGet()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				seeded := make([]*dao.Job, 0, len(testCase.fixtures))

				for _, fixture := range testCase.fixtures {
					entity, err := daoJobEnqueue.Exec(ctx, fixture)
					require.NoError(t, err)

					seeded = append(seeded, entity)
				}

				result, err := daoJobGet.Exec(ctx, testCase.request)
				require.ErrorIs(t, err, testCase.expectErr)

				if testCase.expectFixture < 0 {
					require.Nil(t, result)

					return
				}

				require.Equal(t, seeded[testCase.expectFixture], result)
			})
		})
	}
}
