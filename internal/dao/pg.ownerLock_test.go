package dao_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun/driver/pgdriver"

	"github.com/a-novel-kit/golib/postgres"
	"github.com/a-novel-kit/golib/postgres/postgrestest"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

func TestPgOwnerLock(t *testing.T) {
	t.Parallel()

	// pgLockNotAvailable is the SQLSTATE of a lock wait past lock_timeout.
	const pgLockNotAvailable = "55P03"

	testCases := []struct {
		name string

		// held is the owner whose lock another transaction holds.
		held    uuid.UUID
		request *dao.OwnerLockRequest

		expectBlocked bool
	}{
		{
			name: "Success",

			held:    otherOwnerID,
			request: &dao.OwnerLockRequest{OwnerID: ownerID},
		},
		{
			name: "Success/SameOwnerWaits",

			held:    ownerID,
			request: &dao.OwnerLockRequest{OwnerID: ownerID},

			expectBlocked: true,
		},
	}

	daoOwnerLock := dao.NewPgOwnerLock()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			postgrestest.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
				t.Helper()

				db, err := postgres.GetContext(ctx)
				require.NoError(t, err)

				holder, err := db.BeginTx(ctx, nil)
				require.NoError(t, err)
				t.Cleanup(func() { _ = holder.Rollback() })

				err = daoOwnerLock.Exec(postgres.WithTx(ctx, holder), &dao.OwnerLockRequest{OwnerID: testCase.held})
				require.NoError(t, err)

				waiter, err := db.BeginTx(ctx, nil)
				require.NoError(t, err)
				t.Cleanup(func() { _ = waiter.Rollback() })

				// The waiter gives up after a short wait instead of waiting for the holder to end.
				_, err = waiter.NewRaw("SET LOCAL lock_timeout = '100ms'").Exec(ctx)
				require.NoError(t, err)

				err = daoOwnerLock.Exec(postgres.WithTx(ctx, waiter), testCase.request)
				if !testCase.expectBlocked {
					require.NoError(t, err)

					return
				}

				var pgErr pgdriver.Error

				require.ErrorAs(t, err, &pgErr)
				require.Equal(t, pgLockNotAvailable, pgErr.Field('C'))
			})
		})
	}
}
