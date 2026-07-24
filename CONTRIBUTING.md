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

## Outbound calls

Every call to an outside provider runs on the one client `cmd` builds from `lib.NewHTTPClient` and injects downward. The linter refuses `http.DefaultClient`, `http.DefaultTransport` and the `http.Get`/`Head`/`Post`/`PostForm` helpers, which are unsized, untraced, and take no context.

Two of that client's settings look like mistakes and are not. **Both timeouts are zero**: a non-streaming model call sends no response headers until generation finishes, so any value there kills exactly the long calls the client exists to carry, and only those — which means it passes CI and fails in production. Deadlines come from the caller's context instead. **`HTTP_CLIENT_MAX_IDLE_CONNS_PER_HOST` must be at least the number of jobs the worker runs at once**, because Go's default is two and every concurrent call past the limit pays a fresh TLS handshake against a host the process is already connected to.

Exercise an outbound call against `daotest.NewProviderServer`, and reach it through `lib.NewHTTPClient` rather than a bare client, so the test runs on the transport the service actually uses. The server replays scripted responses in order and records what it received. Its `Hang` and `Drop` options cover the two failures a recorded fixture cannot reproduce — a provider that accepts a request and goes quiet, and one that drops the connection without answering. Response bodies worth reading live in `testdata/` as pretty-printed JSON and load with `daotest.Golden`, so a change to one reads in review as the lines that changed.

---

## Test-support packages

A fixture shared across packages cannot live in a `_test.go` file, because Go excludes those from a package's exported surface. It goes in a regular file in a `<layer>test` package beside the layer it supports — `internal/config/configtest`, `internal/dao/daotest`.

Those packages compile into the module, so their boundary is a rule review enforces rather than one the compiler does: **production code never imports them.** They ride into the Docker build context with the rest of their layer's directory, and are never linked, because no `cmd` reaches them.

---

## Questions?

[Open an issue](https://github.com/a-novel/service-narrative-engine/issues) — include logs and environment details.
