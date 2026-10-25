package handlers_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/a-novel-kit/golib/postgres"
	postgrespresets "github.com/a-novel-kit/golib/postgres/presets"

	"github.com/a-novel/service-narrative-engine/internal/config/configtest"
)

// probeSetup selects the state of the PostgreSQL connection a health probe meets.
type probeSetup int

const (
	// probeHealthy gives the probe an open connection.
	probeHealthy probeSetup = iota
	// probeWithoutPostgres leaves PostgreSQL out of the context.
	probeWithoutPostgres
	// probeClosedPostgres gives the probe a closed connection pool.
	probeClosedPostgres
	// probeInTransaction runs the probe inside an open transaction, which it must leave open.
	probeInTransaction
	// probeCancelled gives the probe a cancelled context.
	probeCancelled
)

// newProbeContext returns the context a health probe runs with under the given setup. Each call
// opens a pool of its own, so closing it leaves the other tests' connections alone.
func newProbeContext(t *testing.T, setup probeSetup) context.Context {
	t.Helper()

	ctx := t.Context()
	if setup == probeWithoutPostgres {
		return ctx
	}

	ctx, err := postgres.NewContext(ctx, postgrespresets.NewDefault(configtest.PostgresPreset.Options()...))
	require.NoError(t, err)

	pg, err := postgres.GetContext(ctx)
	require.NoError(t, err)

	db, ok := pg.(*bun.DB)
	require.True(t, ok)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	switch setup {
	case probeClosedPostgres:
		require.NoError(t, db.Close())
	case probeInTransaction:
		// The transaction outlives the test context, so the rollback proves the probe left it open.
		tx, err := db.BeginTx(context.WithoutCancel(ctx), nil)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, tx.Rollback()) })

		ctx = context.WithValue(ctx, postgres.ContextKey{}, tx)
	case probeCancelled:
		cancelled, cancel := context.WithCancel(ctx)
		cancel()

		ctx = cancelled
	}

	return ctx
}
