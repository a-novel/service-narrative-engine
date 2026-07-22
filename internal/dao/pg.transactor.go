package dao

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/uptrace/bun"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

// A Transactor is the PostgreSQL implementation of the transaction scope the business layer
// declares.
//
// It lives here rather than in the shared library only because the epic that introduced it took no
// cross-repo release train. Once the library absorbs these primitives this collapses to a delegate,
// and the semantics below stop being defined in two places:
// https://github.com/a-novel-kit/golib/issues/369
type Transactor struct {
	opts *sql.TxOptions
}

// NewTransactor returns a Transactor opening its transactions with opts. A nil opts leaves the
// database defaults in place, which is read-committed isolation.
//
// A nested call joins the transaction already in progress, so it never reaches opts. An operation
// that depends on a specific isolation level must therefore be the outermost transaction, or it
// will silently run under whatever level its caller chose.
func NewTransactor(opts *sql.TxOptions) *Transactor {
	return &Transactor{opts: opts}
}

func (dao *Transactor) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	ctx, span := otel.Tracer().Start(ctx, "dao.Transactor")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("get database handle: %w", err))
	}

	pool, isPool := db.(*bun.DB)
	if !isPool {
		// The context already carries a transaction rather than the pool, so this call is nested.
		// Join the transaction in progress instead of opening a savepoint: one unit of work has one
		// outcome, and a caller that sees its own commit succeed should not discover later that an
		// inner block rolled part of it back.
		return fn(ctx)
	}

	// The transaction has to be installed on the context the callback receives, because that is how
	// every data-access object in this service resolves its handle. The library's own RunInTx
	// forwards the context unchanged, so a data-access call inside its callback resolves the pool
	// and commits on its own while appearing to take part.
	err = pool.RunInTx(ctx, dao.opts, func(ctx context.Context, tx bun.Tx) error {
		return fn(context.WithValue(ctx, postgres.ContextKey{}, tx))
	})
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("run in transaction: %w", err))
	}

	otel.ReportSuccessNoContent(span)

	return nil
}

// InTx reports whether ctx carries an open transaction rather than the connection pool.
//
// Work that must not hold a pooled connection open — any call to an external service — guards
// itself with this. The guard belongs to the data-access object making the call rather than to the
// client it calls through, because internal/lib sits below internal/dao and cannot reach it.
//
// It reports true under postgres.RunTransactionalTest, whose passthrough transaction is not a
// *bun.DB. A test covering an outbound call must therefore run under postgres.RunDBTest, which puts
// a real pool on the context.
func InTx(ctx context.Context) bool {
	db, err := postgres.GetContext(ctx)
	if err != nil {
		return false
	}

	_, isPool := db.(*bun.DB)

	return !isPool
}
