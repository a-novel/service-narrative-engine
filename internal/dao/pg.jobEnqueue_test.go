package dao_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestJobEnqueue(t *testing.T) {
	t.Parallel()

	ownerAlice := uuid.MustParse("00000000-0000-0000-0000-00000000a11c")
	ownerBob := uuid.MustParse("00000000-0000-0000-0000-00000000b0b0")

	keyOf := func(value string) *string { return &value }

	// A request whose fields the individual cases override. Every field the database requires is set,
	// so a case only states what it is actually about.
	newRequest := func(id uuid.UUID) *dao.JobEnqueueRequest {
		return &dao.JobEnqueueRequest{
			ID:                 id,
			Kind:               "generate",
			Payload:            json.RawMessage(`{"seed":"a idea"}`),
			OwnerID:            ownerAlice,
			IdempotencyKey:     keyOf("key-1"),
			RequestFingerprint: []byte{0x01, 0x02},
			MaxAttempts:        1,
		}
	}

	withID := func(id string, mutate func(*dao.JobEnqueueRequest)) *dao.JobEnqueueRequest {
		request := newRequest(uuid.MustParse(id))
		if mutate != nil {
			mutate(request)
		}

		return request
	}

	testCases := []struct {
		name string

		// requests run in order against one database, so a case describes a sequence of calls rather
		// than a single one. Deduplication is only observable across two of them.
		requests []*dao.JobEnqueueRequest

		// withinRolledBackTx runs the sequence inside a transaction that rolls back.
		withinRolledBackTx bool

		// expectCreated says, per request, whether that call created the row it returned. It is
		// derived the way core will derive it: the returned id equals the one the caller minted.
		expectCreated []bool
		expectRows    int
	}{
		{
			name: "Success",

			requests: []*dao.JobEnqueueRequest{
				withID("00000000-0000-0000-0000-000000000001", nil),
			},

			expectCreated: []bool{true},
			expectRows:    1,
		},
		{
			// The second call attaches to the first job rather than erroring, so a client retrying a
			// timed-out request is never charged for a second generation.
			name: "Success/IdempotentReplay",

			requests: []*dao.JobEnqueueRequest{
				withID("00000000-0000-0000-0000-000000000001", nil),
				withID("00000000-0000-0000-0000-000000000002", nil),
			},

			expectCreated: []bool{true, false},
			expectRows:    1,
		},
		{
			// The regression a two-column unique index would fail: without kind in the key, the
			// second call is served as a replay of the first and the consolidate job never runs.
			name: "Success/SameKeyDifferentKinds",

			requests: []*dao.JobEnqueueRequest{
				withID("00000000-0000-0000-0000-000000000001", nil),
				withID("00000000-0000-0000-0000-000000000002", func(r *dao.JobEnqueueRequest) {
					r.Kind = "consolidate"
				}),
			},

			expectCreated: []bool{true, true},
			expectRows:    2,
		},
		{
			// Keys are scoped per owner, so one owner's choice of key cannot collide with another's.
			name: "Success/SameKeyDifferentOwners",

			requests: []*dao.JobEnqueueRequest{
				withID("00000000-0000-0000-0000-000000000001", nil),
				withID("00000000-0000-0000-0000-000000000002", func(r *dao.JobEnqueueRequest) {
					r.OwnerID = ownerBob
				}),
			},

			expectCreated: []bool{true, true},
			expectRows:    2,
		},
		{
			// The unique index is partial, so null keys are not compared against each other and a
			// kind cheap enough to run twice can skip deduplication entirely.
			name: "Success/WithoutIdempotencyKey",

			requests: []*dao.JobEnqueueRequest{
				withID("00000000-0000-0000-0000-000000000001", func(r *dao.JobEnqueueRequest) {
					r.IdempotencyKey = nil
				}),
				withID("00000000-0000-0000-0000-000000000002", func(r *dao.JobEnqueueRequest) {
					r.IdempotencyKey = nil
				}),
			},

			expectCreated: []bool{true, true},
			expectRows:    2,
		},
		{
			// The enqueue resolves its handle from the context, so it joins a caller's unit of work
			// instead of committing on its own. Rolling that unit back leaves no job behind.
			name: "Success/RolledBackTransaction",

			requests: []*dao.JobEnqueueRequest{
				withID("00000000-0000-0000-0000-000000000001", nil),
			},

			withinRolledBackTx: true,

			expectCreated: []bool{true},
			expectRows:    0,
		},
	}

	daoJobEnqueue := dao.NewJobEnqueue()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgres.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				// run issues the sequence and checks each call against its expected creator verdict.
				// It takes the context it is given so the same body serves the committed and the
				// rolled-back case.
				run := func(ctx context.Context) {
					var winner *dao.Job

					for index, request := range testCase.requests {
						entity, err := daoJobEnqueue.Exec(ctx, request)
						require.NoError(t, err)

						require.Equal(t, testCase.expectCreated[index], entity.ID == request.ID,
							"request %d: unexpected creator verdict", index)

						// A losing call must return the row the winning one created, not a fresh one.
						if !testCase.expectCreated[index] {
							require.NotNil(t, winner)
							require.Equal(t, winner.ID, entity.ID)
							require.Equal(t, winner.CreatedAt, entity.CreatedAt)
						} else {
							winner = entity
						}

						require.Equal(t, request.OwnerID, entity.OwnerID)
						require.Equal(t, dao.JobStatusPending, entity.Status)
						require.Nil(t, entity.SettledAt)
						require.Nil(t, entity.LeaseExpiresAt)
					}
				}

				if testCase.withinRolledBackTx {
					errRollback := errors.New("rollback")

					err := postgres.NewTransactor(nil).WithinTx(ctx, func(ctx context.Context) error {
						run(ctx)

						return errRollback
					})
					require.ErrorIs(t, err, errRollback)
				} else {
					run(ctx)
				}

				db, err := postgres.GetContext(ctx)
				require.NoError(t, err)

				count, err := db.NewSelect().Model((*dao.Job)(nil)).Count(ctx)
				require.NoError(t, err)
				require.Equal(t, testCase.expectRows, count)
			})
		})
	}
}
