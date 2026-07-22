# Contributing to service-narrative-engine

This file covers only what is specific to this service. For service-level contribution shared across every service — the architecture, the layers, the conventions — start with the [service & architecture concepts](https://github.com/a-novel/.github/blob/master/CONTRIBUTING.md). Platform setup and day-to-day commands are in the [developer onboarding guide](https://github.com/a-novel-kit/.github/blob/master/README.md).

`service-narrative-engine` currently ships a placeholder `item` resource: a dummy entity that implements the common service contracts end to end, with no real feature of its own yet. It is the scaffold the narrative-engine domain will grow into.

---

## Running it locally

Start the server and load its ports into your shell:

```bash
a-novel run start service-narrative-engine/rest
eval "$(a-novel run env service-narrative-engine)"
```

Check it is alive:

```bash
curl http://localhost:${SERVICE_NARRATIVE_ENGINE_REST_PORT}/ping          # REST liveness
curl http://localhost:${SERVICE_NARRATIVE_ENGINE_REST_PORT}/healthcheck   # REST: Postgres dependency
```

The `item` CRUD routes (`/items`, `/item`) are placeholder wiring, not a feature; their request/response shapes live in [`openapi.yaml`](./openapi.yaml).

---

## Transactions

Two or more writes that must land together are wrapped in a `transaction.Transactor`, taken as a constructor dependency by any service that needs one and injected in `cmd`. It names no database, so business code says "these writes are one unit" without knowing what stores them:

```go
err := service.transactor.WithinTx(ctx, func(ctx context.Context) error {
	// every data-access call made with this ctx is part of one transaction
})
```

**Pass the callback's `ctx` down, not the outer one.** Data-access objects resolve their database handle from the context, and the transaction is installed on the context the callback receives. An inner call given the outer context runs on the connection pool and commits on its own, while the surrounding block still reports success.

Two rules follow, and the shared library's documentation is the contract for both:

- **Never call an external service inside `WithinTx`.** An open transaction holds a pooled connection for its whole lifetime; pinning one for the length of a third-party call exhausts the pool and blocks vacuuming. Persist what the call needs, close the transaction, make the call, then open a new transaction to record the result. `postgres.InTx(ctx)` reports whether a transaction is open, so a data-access object that makes an outbound call can refuse rather than rely on the convention holding.
- **A nested `WithinTx` joins the transaction in progress**, so a rollback anywhere discards the whole outermost unit of work — including work the outer caller believed was already safe. Nesting is legal; it should be deliberate. A nested call also never sees its own `sql.TxOptions`, so an operation needing a specific isolation level has to be the outermost transaction.

Unit-test a service that takes a transactor with `transactiontest.NewTransactor`, which runs the callback inline, or `NewFailingTransactor` to cover the path where the unit of work never opens. A test that needs a real rollback needs a real database: use `postgres.RunDBTest`, never `RunTransactionalTest`, whose passthrough transaction cannot tell a working transactor from a broken one.

---

## Schema conventions

These hold for every new table. The scaffold `items` table predates some of them and is not the reference.

**Identifiers are time-ordered and minted in Go.** Columns default to `uuidv7()`, which the project's PostgreSQL 18 image provides natively, so a table under insert churn keeps index locality instead of scattering writes across the whole B-tree. Core still generates the identifier and passes it in: an insert that has to read back a database-generated id cannot tell its own row from one a concurrent caller inserted under the same unique key.

**Timestamps are full precision, and the database is the clock.** Declare `timestamptz`, never `timestamp(0)` — second precision cannot order two commits, let alone express a lease expiry. Default them to `clock_timestamp()`, never `now()` or `CURRENT_TIMESTAMP`: those two are frozen at transaction start, so a column written inside a transaction can never advance past a value its neighbors already hold. Where several services or workers compare timestamps, the database has to be the single clock, or application-server skew enters the arithmetic.

**Owned rows carry an `owner_id`, with no cross-service foreign key.** The column holds the acting user from the verified token, never a value from the request payload. Identity belongs to another service, so there is nothing local to reference and no constraint to declare — the token is what makes the value trustworthy.

**Ownership is a query predicate, not a check after the fact:**

```sql
SELECT
  *
FROM
  jobs
WHERE
  id = ?0
  AND owner_id = ?1;
```

A predicate is fail-closed: a caller that forgets the owner argument fails to scan rather than returning someone else's row, whereas a later `if row.OwnerID != actor.UserID` is one early return away from being skipped. It also collapses "no such row" and "not your row" into one no-rows result, which removes an existence oracle over a priced resource at no cost.

**A cross-owner read is not-found, never access-denied.** The data-access object joins `sql.ErrNoRows` onto its own sentinel and the handler maps that to 404. Answering 403 would confirm the row exists.

**Every migration gets its own deliberately allocated prefix.** Take it from `date '+%Y%m%d%H%M%S'` at the moment you create the file. bun derives a migration's identity from that numeric prefix alone, so two files sharing one merge into a single migration: the second replaces the first, with no error at discovery and none at apply, and the first migration simply never runs. The `test-go` job checks prefixes for uniqueness per direction before it applies anything — an `.up.sql` and its `.down.sql` are meant to share a prefix, two different migrations are not.

---

## Questions?

[Open an issue](https://github.com/a-novel/service-narrative-engine/issues) — include logs and environment details.
