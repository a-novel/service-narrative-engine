package core

import "context"

// A Transactor runs a unit of work atomically: every persistence call made with the context it
// hands the callback either commits together or is rolled back together.
//
// The callback receives no transaction handle. That is deliberate, and it is the reason this port
// exists rather than the business layer calling the database library directly: a handle callers are
// expected to thread through every nested call is a handle they can forget, and forgetting it fails
// silently — the calls run outside the transaction, the block commits, and the tests pass. With
// nothing to thread, the mistake cannot be written.
//
// Two consequences are worth knowing before using it.
//
// A nested WithinTx joins the transaction already in progress instead of opening a nested one, so a
// rollback anywhere discards the whole outermost unit of work. An inner operation that treats a
// failure as locally recoverable is therefore discarding its caller's work too. Nesting is legal,
// but it should be a deliberate choice rather than an accident of composition.
//
// Nothing that talks to an external service may run inside the callback. A transaction holds a
// pooled database connection for as long as it is open, and pinning one for the length of a call to
// a third party exhausts the pool and blocks vacuuming across the database. Persist what the
// external call needs, close the transaction, make the call, then open a new transaction to record
// the result.
type Transactor interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}
