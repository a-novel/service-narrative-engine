# Service Narrative Engine

An A-Novel backend service. It currently ships a placeholder `item` resource — a named entity with an optional description, exposed through full CRUD — that exercises the platform's common service contracts end to end while the real narrative-engine domain is built out.

[![X (formerly Twitter) Follow](https://img.shields.io/twitter/follow/agorastoryverse)](https://twitter.com/agorastoryverse)
[![Discord](https://img.shields.io/discord/1315240114691248138?logo=discord)](https://discord.gg/rp4Qr8cA)

<hr />

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/a-novel/service-narrative-engine)
![GitHub repo file or directory count](https://img.shields.io/github/directory-file-count/a-novel/service-narrative-engine)
![GitHub code size in bytes](https://img.shields.io/github/languages/code-size/a-novel/service-narrative-engine)

![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/a-novel/service-narrative-engine/main.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/a-novel/service-narrative-engine)](https://goreportcard.com/report/github.com/a-novel/service-narrative-engine)
[![codecov](https://codecov.io/gh/a-novel/service-narrative-engine/graph/badge.svg)](https://codecov.io/gh/a-novel/service-narrative-engine)

![Coverage graph](https://codecov.io/gh/a-novel/service-narrative-engine/graphs/sunburst.svg)

## What it does

The service's only domain object today is `item` — a named entity with an optional description — exposed through full CRUD. It demonstrates the [layered architecture](https://github.com/a-novel/.github/blob/master/CONTRIBUTING.md) (DAO → core → handler) and the client packages a real service inherits.

The service exposes a **public REST API** (`cmd/rest`) — `/ping`, `/healthcheck`, and the `/items` + `/item` CRUD routes — for any HTTP client.

## Deploying

The service runs as published OCI images plus a PostgreSQL database. The server is stateless, so it scales to as many replicas as you need behind a load balancer; all state lives in Postgres.

> **OpenTofu modules are the planned canonical deployment path.** Until they land, deploy the images with any container orchestrator — the composition below is the reference for which images to run, how they wire together, and the environment they expect.

| Image                                      | Role                                                                        |
| ------------------------------------------ | --------------------------------------------------------------------------- |
| `service-narrative-engine/rest`            | Public item CRUD + health API.                                              |
| `service-narrative-engine/jobs/migrations` | One-shot schema migration job; runs to completion before the servers start. |
| `service-narrative-engine/database`        | Pre-tuned PostgreSQL image — or bring your own Postgres.                    |

Pin every image to the same release tag. A production deployment runs `database`, then `migrations` to completion, then any number of `rest` replicas:

<!-- TODO(project-docs): replace v0.0.0 with the service's release tag -->

```yaml
services:
  postgres-narrative-engine:
    image: ghcr.io/a-novel/service-narrative-engine/database:v0.0.0
    networks: [api]
    environment:
      POSTGRES_PASSWORD: postgres
      POSTGRES_USER: postgres
      POSTGRES_DB: postgres
      POSTGRES_HOST_AUTH_METHOD: scram-sha-256
      POSTGRES_INITDB_ARGS: --auth=scram-sha-256
    volumes:
      - narrative-engine-postgres-data:/var/lib/postgresql/

  migrations-narrative-engine:
    image: ghcr.io/a-novel/service-narrative-engine/jobs/migrations:v0.0.0
    depends_on:
      postgres-narrative-engine: { condition: service_healthy }
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-narrative-engine:5432/postgres?sslmode=disable"
    networks: [api]

  service-narrative-engine:
    image: ghcr.io/a-novel/service-narrative-engine/rest:v0.0.0
    ports: ["${SERVICE_NARRATIVE_ENGINE_REST_PORT}:8080"] # the container always listens on 8080
    depends_on:
      postgres-narrative-engine: { condition: service_healthy }
      migrations-narrative-engine: { condition: service_completed_successfully }
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-narrative-engine:5432/postgres?sslmode=disable"
    networks: [api]

networks:
  api:

volumes:
  narrative-engine-postgres-data:
```

### Configuration

Every variable is read from the process environment. Env-var names can be globally prefixed via `SERVICE_NARRATIVE_ENGINE_ENV_PREFIX`.

| Name           | Description                                 | Images |
| -------------- | ------------------------------------------- | ------ |
| `POSTGRES_DSN` | PostgreSQL connection string. **Required.** | all    |

<details>
<summary>Optional configuration (REST tuning, OpenTelemetry)</summary>

REST tuning (images `rest`, `standalone-rest`):

| Name                          | Description                          | Default          |
| ----------------------------- | ------------------------------------ | ---------------- |
| `REST_MAX_REQUEST_SIZE`       | Maximum request body size, in bytes. | `2097152` (2MiB) |
| `REST_TIMEOUT_READ`           | Read timeout.                        | `15s`            |
| `REST_TIMEOUT_READ_HEADER`    | Header read timeout.                 | `3s`             |
| `REST_TIMEOUT_WRITE`          | Write timeout.                       | `30s`            |
| `REST_TIMEOUT_IDLE`           | Idle keep-alive timeout.             | `60s`            |
| `REST_TIMEOUT_REQUEST`        | Per-request timeout.                 | `60s`            |
| `REST_CORS_ALLOWED_ORIGINS`   | CORS allowed origins.                | `*`              |
| `REST_CORS_ALLOWED_HEADERS`   | CORS allowed headers.                | `*`              |
| `REST_CORS_ALLOW_CREDENTIALS` | CORS allow-credentials flag.         | `false`          |
| `REST_CORS_MAX_AGE`           | CORS max-age, in seconds.            | `3600`           |

Logs and tracing — OpenTelemetry supports a stdout and a Google Cloud exporter (all server images):

| Name                | Description                                                           | Default                    |
| ------------------- | --------------------------------------------------------------------- | -------------------------- |
| `OTEL`              | Enable OTel tracing; the variables below pick the exporter.           | `false`                    |
| `GCLOUD_PROJECT_ID` | Google Cloud project ID. When set, switches the OTel exporter to GCP. |                            |
| `APP_NAME`          | Application name attached to traces and logs.                         | `service-narrative-engine` |

</details>

## Using the client package

A REST client ships with the service. The snippet below is the **minimum viable call**; the full surface is what your editor's intellisense and the [API reference](https://a-novel.github.io/service-narrative-engine) are for.

### JavaScript / TypeScript (REST)

The package is published to GitHub Packages, which requires a Personal Access Token with the `read:packages` scope even for public packages ([why](https://github.com/orgs/community/discussions/23386#discussioncomment-3240193)). Add to `.npmrc` (project root or `$HOME`):

```ini
@a-novel:registry=https://npm.pkg.github.com
@a-novel-kit:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=${YOUR_PERSONAL_ACCESS_TOKEN}
```

```bash
pnpm add @a-novel/service-narrative-engine-rest
```

```typescript
import { NarrativeEngineApi, itemCreate, itemList } from "@a-novel/service-narrative-engine-rest";

const api = new NarrativeEngineApi("http://service-narrative-engine:8080");

const created = await itemCreate(api, "My Item", "An optional description.");
const items = await itemList(api, 10, 0);
```

API reference: [a-novel.github.io/service-narrative-engine](https://a-novel.github.io/service-narrative-engine).

## Running locally

For a throwaway instance without the dev toolchain, the **standalone** image bundles the server and migrations in one container. It runs migrations on every boot — handy for a quick spin-up, unsafe under multi-replica production restarts.

```yaml
services:
  postgres-narrative-engine:
    image: ghcr.io/a-novel/service-narrative-engine/database:v0.0.0
    networks: [api]
    environment:
      POSTGRES_PASSWORD: postgres
      POSTGRES_USER: postgres
      POSTGRES_DB: postgres
      POSTGRES_HOST_AUTH_METHOD: scram-sha-256
      POSTGRES_INITDB_ARGS: --auth=scram-sha-256

  service-narrative-engine:
    image: ghcr.io/a-novel/service-narrative-engine/standalone-rest:v0.0.0
    ports: ["${SERVICE_NARRATIVE_ENGINE_REST_PORT}:8080"]
    depends_on:
      postgres-narrative-engine: { condition: service_healthy }
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-narrative-engine:5432/postgres?sslmode=disable"
    networks: [api]

networks:
  api:
```

Working on the service itself? Use the `a-novel` CLI (`a-novel run start service-narrative-engine/rest`) instead — see [CONTRIBUTING](./CONTRIBUTING.md).

## Contributing

Platform setup and the day-to-day commands live in the [developer onboarding guide](https://github.com/a-novel-kit/.github/blob/master/README.md). Service-specific concepts and local interactions are in [CONTRIBUTING.md](./CONTRIBUTING.md).
