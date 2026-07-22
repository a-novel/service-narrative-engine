package dao_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/a-novel-kit/golib/postgres"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
	"github.com/a-novel/service-narrative-engine/internal/dao"
	"github.com/a-novel/service-narrative-engine/internal/models/migrations"
)

// errTransactorTest is the failure a test callback returns to trigger a rollback. It stands for any
// error a real unit of work might raise; only its identity matters.
var errTransactorTest = errors.New("rollback")

// writeItem inserts one item through the ordinary data-access path, so what the tests below observe
// is a real data-access object resolving its own handle from the context rather than a hand-written
// statement that could be threaded differently.
func writeItem(ctx context.Context, t *testing.T, id uuid.UUID) {
	t.Helper()

	_, err := dao.NewItemCreate().Exec(ctx, &dao.ItemCreateRequest{
		ID:   id,
		Name: "transactor test item",
		Now:  time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
}

// requireItemAbsent asserts the row never reached the database, which is only true if the
// transaction that wrote it was rolled back.
func requireItemAbsent(ctx context.Context, t *testing.T, id uuid.UUID) {
	t.Helper()

	_, err := dao.NewItemGet().Exec(ctx, &dao.ItemGetRequest{ID: id})
	require.ErrorIs(t, err, dao.ErrItemGetNotFound)
}

func requireItemPresent(ctx context.Context, t *testing.T, id uuid.UUID) {
	t.Helper()

	_, err := dao.NewItemGet().Exec(ctx, &dao.ItemGetRequest{ID: id})
	require.NoError(t, err)
}

// TestTransactorRollback is the regression test for the whole task: it fails against a transactor
// that leaves the pool on the context, because the item DAO would then commit on its own and the
// row would survive the rollback.
func TestTransactorRollback(t *testing.T) {
	t.Parallel()

	id := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	postgres.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
		t.Helper()

		err := dao.NewTransactor(nil).WithinTx(ctx, func(ctx context.Context) error {
			writeItem(ctx, t, id)

			return errTransactorTest
		})
		require.ErrorIs(t, err, errTransactorTest)

		requireItemAbsent(ctx, t, id)
	})
}

// TestTransactorRollbackLibraryPath pins the behaviour the transactor exists to avoid. The library
// helper opens a real transaction but hands the callback the original context, so the item DAO
// resolves the pool, commits independently, and the row outlives the rollback.
//
// It asserts the row SURVIVES. That is not a mistake: the day this test starts failing, the library
// has been fixed and the local transactor can delegate to it.
func TestTransactorRollbackLibraryPath(t *testing.T) {
	t.Parallel()

	id := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	postgres.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
		t.Helper()

		err := postgres.RunInTx(ctx, nil, func(ctx context.Context, _ bun.IDB) error {
			writeItem(ctx, t, id)

			return errTransactorTest
		})
		require.ErrorIs(t, err, errTransactorTest)

		requireItemPresent(ctx, t, id)
	})
}

// TestTransactorNested covers the joining rule: an inner rollback discards the outer unit of work,
// because the inner call takes part in the transaction already in progress rather than opening a
// savepoint it could roll back alone.
func TestTransactorNested(t *testing.T) {
	t.Parallel()

	outerID := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	innerID := uuid.MustParse("00000000-0000-0000-0000-000000000004")

	postgres.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
		t.Helper()

		transactor := dao.NewTransactor(nil)

		err := transactor.WithinTx(ctx, func(ctx context.Context) error {
			writeItem(ctx, t, outerID)

			return transactor.WithinTx(ctx, func(ctx context.Context) error {
				writeItem(ctx, t, innerID)

				return errTransactorTest
			})
		})
		require.ErrorIs(t, err, errTransactorTest)

		requireItemAbsent(ctx, t, outerID)
		requireItemAbsent(ctx, t, innerID)
	})
}

// TestTransactorCommit is the counterpart to the rollback tests: a callback that returns no error
// commits, so the transactor is not merely discarding everything.
func TestTransactorCommit(t *testing.T) {
	t.Parallel()

	id := uuid.MustParse("00000000-0000-0000-0000-000000000005")

	postgres.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
		t.Helper()

		err := dao.NewTransactor(nil).WithinTx(ctx, func(ctx context.Context) error {
			writeItem(ctx, t, id)

			return nil
		})
		require.NoError(t, err)

		requireItemPresent(ctx, t, id)
	})
}

func TestInTx(t *testing.T) {
	t.Parallel()

	postgres.RunDBTest(t, configtest.PostgresPreset, migrations.Migrations, func(ctx context.Context, t *testing.T) {
		t.Helper()

		require.False(t, dao.InTx(ctx), "the pool is on the context outside a transaction")

		err := dao.NewTransactor(nil).WithinTx(ctx, func(ctx context.Context) error {
			require.True(t, dao.InTx(ctx), "the transaction is on the context inside WithinTx")

			return nil
		})
		require.NoError(t, err)
	})
}

// TestInTxNoContext covers the case where nothing seeded the context at all. InTx answers "is a
// transaction open", and no database handle at all is not one.
func TestInTxNoContext(t *testing.T) {
	t.Parallel()

	require.False(t, dao.InTx(t.Context()))
}
