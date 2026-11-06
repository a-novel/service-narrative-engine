# Contributing to service-narrative-engine

This document covers what is specific to the Narrative Engine service. Read the [README](./README.md) first for what the service does. The [developer guide](https://a-novel-kit.github.io/stack/) sets up the platform, and the [taxonomy](https://github.com/a-novel/.github/blob/master/docs/taxonomy.md#inside-a-service-layers-and-contracts) explains the layers every service shares.

---

## Quick local interactions

Start the server, then load the ports the `a-novel` daemon gave it into your shell:

```bash
a-novel run start service-narrative-engine/rest
eval "$(a-novel run env service-narrative-engine)"
```

Check that it answers:

```bash
# Liveness
curl "http://localhost:${REST_PORT}/v1/ping"

# Dependency check (Postgres and JSON Keys)
curl "http://localhost:${REST_PORT}/v1/healthcheck"
```

The routes are described in [`openapi.yaml`](./openapi.yaml).

---

## Transactions

Writes that must land together run in a `transaction.Transactor`. The service that needs one takes it as a constructor dependency, and `cmd` injects it. The transactor names no database, so business code says "these writes are one unit" without knowing what stores them:

```go
// internal/core
type SomeService struct {
	dao        SomeServiceDao
	transactor transaction.Transactor
}

err := service.transactor.WithinTx(ctx, func(ctx context.Context) error {
	// every data-access call made with this ctx is part of one transaction
})

// cmd
service := core.NewSomeService(daoSomething, postgres.NewTransactor(nil))
```

**Pass the callback's `ctx` down, never the outer one.** A data-access object resolves its database handle from the context, and the transaction lives on the context the callback receives. An inner call given the outer context runs on the connection pool and commits on its own, while the block still reports success.

The golib [`transaction` package](https://pkg.go.dev/github.com/a-novel-kit/golib/transaction) holds two more rules. Call no external service inside `WithinTx`. A nested `WithinTx` joins the transaction in progress, so a rollback anywhere discards the whole unit of work. `postgres.InTx(ctx)` reports whether a transaction is open, so a data-access object that makes an outbound call can refuse to run inside one.

Unit-test a service that takes a transactor with `transactiontest.NewTransactor`, which runs the callback inline. `NewFailingTransactor` covers the path where the transaction never opens: asserting that no dependency is reached proves the writes sit inside the transaction. A test that needs a real rollback needs a real database. Use `postgrestest.RunDBTest` for it, since the passthrough transaction of `postgrestest.RunTransactionalTest` cannot tell a working transactor from a broken one.

---

## Code license

The repository's copyrightable source code owned by the project licensor is released under [RPL-1.5](LICENSE). See [NOTICE.md](NOTICE.md) for the current licensor, source-file notice, and history of licensing.

## Contributor agreement

Before submitting a contribution for possible inclusion in an official A-Novel repository, please review [CLA.md](CLA.md). The project maintainer may require a duly signed individual or corporate Contributor License Agreement before accepting the contribution. A pull request, checkbox, or signed-off commit by itself does not replace a legally valid CLA signature.

If your contribution was developed for an employer or customer, or includes third-party material, ensure that you have permission to grant the rights in the CLA and disclose any third-party license terms.

The CLA gives the maintainer additional rights to use accepted contributions, including proprietary relicensing where lawful. It does not change the RPL-1.5 license of public releases or bind independent fork maintainers who have not signed it.

## Submitting changes

1. Open an issue or pull request describing the proposed change.
2. Identify any incorporated third-party material and applicable notices.
3. Where requested, complete a signed CLA and identify the relevant GitHub identity, repository and PR/commit.
4. Maintain copyright attribution and the RPL-1.5 Exhibit A notices required by the repository license.

Project maintainers will separately record signatures and match them to accepted PRs or commits. GitHub does not automatically execute or validate this agreement merely because the file exists in the repository.

---

## Questions?

[Open an issue](https://github.com/a-novel/service-narrative-engine/issues) with logs and environment details.
