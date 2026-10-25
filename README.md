# Service Template

The scaffold every new A-Novel backend service starts from, with an example `item` resource built through every layer.

[![X (formerly Twitter) Follow](https://img.shields.io/twitter/follow/agorastoryverse)](https://twitter.com/agorastoryverse)
[![Discord](https://img.shields.io/discord/1315240114691248138?logo=discord)](https://discord.gg/rp4Qr8cA)

<hr />

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/a-novel/service-template)
![GitHub repo file or directory count](https://img.shields.io/github/directory-file-count/a-novel/service-template)
![GitHub code size in bytes](https://img.shields.io/github/languages/code-size/a-novel/service-template)

![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/a-novel/service-template/main.yaml)
[![codecov](https://codecov.io/gh/a-novel/service-template/graph/badge.svg)](https://codecov.io/gh/a-novel/service-template)

![Coverage graph](https://codecov.io/gh/a-novel/service-template/graphs/sunburst.svg)

## Using this template

Fork this repository to start a service. The rest of this README is the standard service README, written for the placeholder `item` resource. Your fork keeps its shape. The [taxonomy](https://github.com/a-novel/.github/blob/master/docs/taxonomy.md#inside-a-service-layers-and-contracts) explains the layers every service shares.

**1. Rename the service.** Set your service's name on the first line, then replace every `service-template` reference.

```bash
SERVICE="service-<name>"

git grep -lz service-template | xargs -0 perl -pi -e "s/service-template/$SERVICE/g"
```

The replace covers the module path and import paths, image names and references, badges and links, and both `package.json` files. Rename by hand what it leaves, where `git grep` finds every use:

| What                | Where                                            | From → to                                                                               |
| ------------------- | ------------------------------------------------ | --------------------------------------------------------------------------------------- |
| Go package          | `generate.go`, `pkg/go/`                         | `servicetemplate` → `service<name>`                                                     |
| Env-var prefix      | `internal/config/env/env.go`                     | `SERVICE_TEMPLATE_ENV_PREFIX` → `SERVICE_<NAME>_ENV_PREFIX`                             |
| Protobuf package    | `internal/models/proto/anovel/template/v0`       | `anovel.template.v0` → `anovel.<name>.v0`, with its folder and the `templatev0` imports |
| JS client class     | `pkg/js/rest/src/`                               | `TemplateApi` → `<Name>Api`                                                             |
| Display name        | `openapi.yaml`, `openapi.html`, both `README.md` | `Service Template` → your service's title                                               |
| Local compose names | `builds/*.yaml`                                  | `template` → `<name>` in container, network and volume names                            |

**2. Replace the `item` resource.** Swap `item` and `Item` for your own resource in each layer, then adjust the fields.

| Layer     | Files                                                                                                                                   |
| --------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| Migration | `internal/models/migrations/20250306000000_items_table.{up,down}.sql`, renamed with `date '+%Y%m%d%H%M%S'`, and its `testdata/` fixture |
| DAO       | `internal/dao/pg.item*.{go,sql}` and their tests                                                                                        |
| Core      | `internal/core/item*.go`, `common.go` and their tests                                                                                   |
| Handlers  | `internal/handlers/rest.item*.go`, `grpc.item*.go` and their tests                                                                      |
| Proto     | `internal/models/proto/anovel/template/v0/item*.proto`                                                                                  |
| API       | `openapi.yaml`, `pkg/go/client.go` and its test, `pkg/js/rest/src/{index,item}.ts`, `pkg/js/test/rest/src/item.test.ts`                 |
| Wiring    | `cmd/rest/main.go`, `cmd/grpc/main.go`                                                                                                  |

**3. Check the fork.** Start its database, since the migration snapshots regenerate against it. Then regenerate the stubs, mocks and snapshots, and run every check.

```bash
a-novel run service infra start "$SERVICE"
eval "$(a-novel run env "$SERVICE")"

pnpm generate
pnpm format
pnpm lint
a-novel test -y
```

Then delete this section and describe your service in the rest of this README. In [CONTRIBUTING.md](./CONTRIBUTING.md), replace the introduction and the local run, which describe the template.

## What it does

The service stores `item` records, each a name with an optional description, behind create, read, list, update and delete operations. It exists to be replaced: every layer, both APIs and both client packages work end to end, so a fork starts from running code.

The service ships two APIs:

- A **private gRPC API** (`cmd/grpc`), for traffic between services on a private network. It authenticates no caller: network policy, ingress or a service mesh controls access.
- A **public REST API** (`cmd/rest`), for any HTTP client.

## Deploying

The service runs as published OCI images plus a PostgreSQL database. Both servers are stateless, so each scales to as many replicas as you need behind a load balancer; all state lives in Postgres.

Deploy the images with any container orchestrator. The composition below shows which images to run, how they wire together, and the environment they expect.

| Image                              | Role                                                                        |
| ---------------------------------- | --------------------------------------------------------------------------- |
| `service-template/grpc`            | Private item CRUD + status API. Internal network only.                      |
| `service-template/rest`            | Public item CRUD + health API.                                              |
| `service-template/jobs/migrations` | One-shot schema migration job; runs to completion before the servers start. |
| `service-template/database`        | PostgreSQL with pgBackRest, or bring your own Postgres.                     |

Pin every image to the same release tag. A production deployment runs `database`, then `migrations` to completion, then any number of `grpc` and/or `rest` replicas. Provide `POSTGRES_USER`, `POSTGRES_PASSWORD`, and `POSTGRES_DATABASE` through the orchestrator; store the password as a secret.

<!-- TODO(project-docs): replace v0.0.0 with the new service's release tag -->

```yaml
services:
  postgres-template:
    image: ghcr.io/a-novel/service-template/database:v0.0.0
    networks: [api]
    environment:
      POSTGRES_PASSWORD: "${POSTGRES_PASSWORD}"
      POSTGRES_USER: "${POSTGRES_USER}"
      POSTGRES_DB: "${POSTGRES_DATABASE}"
      POSTGRES_HOST_AUTH_METHOD: scram-sha-256
      POSTGRES_INITDB_ARGS: --auth=scram-sha-256
    volumes:
      - template-postgres-data:/var/lib/postgresql/
    # Podman ignores the image's own check, so the dependents' wait needs this one.
    healthcheck:
      test: ["CMD", "pg_isready"]
      interval: 1s
      timeout: 5s
      retries: 10

  migrations-template:
    image: ghcr.io/a-novel/service-template/jobs/migrations:v0.0.0
    depends_on:
      postgres-template: { condition: service_healthy }
    environment:
      POSTGRES_HOST: postgres-template
      POSTGRES_PORT: "5432"
      POSTGRES_USER: "${POSTGRES_USER}"
      POSTGRES_PASSWORD: "${POSTGRES_PASSWORD}"
      POSTGRES_DATABASE: "${POSTGRES_DATABASE}"
      POSTGRES_TLS_ENABLED: "false"
    networks: [api]

  service-template:
    image: ghcr.io/a-novel/service-template/grpc:v0.0.0 # or ghcr.io/a-novel/service-template/rest:v0.0.0
    ports: ["${GRPC_PORT}:8080"] # the container always listens on 8080; map ${REST_PORT} for the rest image
    depends_on:
      postgres-template: { condition: service_healthy }
      migrations-template: { condition: service_completed_successfully }
    environment:
      POSTGRES_HOST: postgres-template
      POSTGRES_PORT: "5432"
      POSTGRES_USER: "${POSTGRES_USER}"
      POSTGRES_PASSWORD: "${POSTGRES_PASSWORD}"
      POSTGRES_DATABASE: "${POSTGRES_DATABASE}"
      POSTGRES_TLS_ENABLED: "false"
    networks: [api]

networks:
  api:

volumes:
  template-postgres-data:
```

Run both servers by adding a second service that reuses the same database and migrations with the `rest` image.

### Database image

The database image packages PostgreSQL and pgBackRest on Wolfi. The container build assembles its [package manifest](./builds/database.apko.yaml) with apko, from [locks](./builds/database.apko.lock.json) that pin every package, so an unchanged image rebuilds to the same digest. Building it takes Docker or Podman alone. The image keeps PostgreSQL's standard `POSTGRES_*` initialization variables, the `uuid-ossp` extension and the volume at `/var/lib/postgresql`. Backup scheduling and repository credentials stay with the operator.

### Configuration

Every variable is read from the process environment. `SERVICE_TEMPLATE_ENV_PREFIX` prepends a prefix to every name, so the service can share an environment without collisions.

| Name                   | Description                                                                                                                   | Images |
| ---------------------- | ----------------------------------------------------------------------------------------------------------------------------- | ------ |
| `POSTGRES_HOST`        | PostgreSQL hostname or IP address. Selects the discrete connection fields. **Required for new deployments.**                  | all    |
| `POSTGRES_PORT`        | PostgreSQL port. Defaults to `5432`.                                                                                          | all    |
| `POSTGRES_USER`        | PostgreSQL login role. **Required when `POSTGRES_HOST` is set.**                                                              | all    |
| `POSTGRES_PASSWORD`    | PostgreSQL login password. **Required when `POSTGRES_HOST` is set; inject it as a secret.**                                   | all    |
| `POSTGRES_DATABASE`    | PostgreSQL database name. **Required when `POSTGRES_HOST` is set.**                                                           | all    |
| `POSTGRES_TLS_ENABLED` | Encrypt the PostgreSQL connection. Defaults to `true`; disable only when another trusted boundary protects the database link. | all    |
| `POSTGRES_DSN`         | Connection-URL fallback, read only when `POSTGRES_HOST` is empty.                                                             | all    |

<details>
<summary>Optional configuration (server tuning, OpenTelemetry)</summary>

gRPC tuning (images `grpc`, `standalone-grpc`):

| Name                    | Description               | Default |
| ----------------------- | ------------------------- | ------- |
| `GRPC_TIMEOUT_SHUTDOWN` | Graceful shutdown budget. | `30s`   |

REST tuning (images `rest`, `standalone-rest`):

| Name                          | Description                                                                                             | Default                          |
| ----------------------------- | ------------------------------------------------------------------------------------------------------- | -------------------------------- |
| `REST_MAX_REQUEST_SIZE`       | Maximum request body size, in bytes.                                                                    | `16384` (16KiB)                  |
| `REST_CLIENT_IP_HEADER`       | Header a trusted proxy sets to the visitor IP. Set it only when every request comes through that proxy. | TCP peer                         |
| `REST_LIVENESS_PATH`          | Ping path, which still answers during a planned downtime and skips the rate limit.                      | `/v0/ping`                       |
| `REST_RATE_LIMITS_ENABLED`    | Rate-limit each visitor on every route but the ping. Turn off only in test stacks.                      | `true`                           |
| `REST_RATE_LIMIT`             | Requests each visitor sends to each route, as `<requests>/<window>`.                                    | `5/1s`                           |
| `REST_TIMEOUT_READ`           | Read timeout.                                                                                           | `15s`                            |
| `REST_TIMEOUT_READ_HEADER`    | Header read timeout.                                                                                    | `3s`                             |
| `REST_TIMEOUT_WRITE`          | Write timeout.                                                                                          | `30s`                            |
| `REST_TIMEOUT_IDLE`           | Idle keep-alive timeout.                                                                                | `60s`                            |
| `REST_TIMEOUT_REQUEST`        | Per-request timeout.                                                                                    | `60s`                            |
| `REST_TIMEOUT_SHUTDOWN`       | Graceful shutdown budget.                                                                               | `30s`                            |
| `REST_CORS_ALLOWED_ORIGINS`   | CORS allowed origins.                                                                                   | `*`                              |
| `REST_CORS_ALLOWED_HEADERS`   | CORS allowed headers.                                                                                   | `*`                              |
| `REST_CORS_ALLOWED_METHODS`   | CORS allowed methods, comma-separated.                                                                  | `HEAD,GET,POST,PUT,PATCH,DELETE` |
| `REST_CORS_ALLOW_CREDENTIALS` | CORS allow-credentials flag.                                                                            | `false`                          |
| `REST_CORS_MAX_AGE`           | CORS max-age, in seconds.                                                                               | `3600`                           |

Database connection pool (server images). The limits are **per process**. The database's `max_connections` has to cover every replica plus the migration job; the stock `postgres` default is 100.

| Name                      | Description                               | Default |
| ------------------------- | ----------------------------------------- | ------- |
| `POSTGRES_MAX_OPEN_CONNS` | Maximum open connections to the database. | `20`    |
| `POSTGRES_MAX_IDLE_CONNS` | Maximum connections kept open while idle. | `20`    |

Planned downtime (server images):

| Name             | Description                                                                                                                                                                                                        | Default |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------- |
| `DOWNTIME_START` | When a planned downtime starts, in RFC 3339. From then until it is removed, even past the announced end, the servers answer `503` / `UNAVAILABLE` except for ping and gRPC health, and start without the database. |         |

Logs and tracing — OpenTelemetry supports a stdout and a Google Cloud exporter (all server images):

| Name                | Description                                                           | Default            |
| ------------------- | --------------------------------------------------------------------- | ------------------ |
| `OTEL`              | Enable OTel tracing; the variables below pick the exporter.           | `false`            |
| `GCLOUD_PROJECT_ID` | Google Cloud project ID. When set, switches the OTel exporter to GCP. |                    |
| `APP_NAME`          | Application name attached to traces and logs.                         | `service-template` |

</details>

## Using the client packages

Two clients ship with the service. Each snippet is the **minimum viable call**. Your editor's intellisense, [pkg.go.dev](https://pkg.go.dev/github.com/a-novel/service-template) and the [API reference](https://a-novel.github.io/service-template) cover the full surface.

- **Go** talks gRPC, for backend services.
- **JavaScript / TypeScript** talks REST, for frontends and Node services.

### Go (gRPC)

```bash
go get github.com/a-novel/service-template
```

```go
package main

import (
	"context"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	servicetemplate "github.com/a-novel/service-template/pkg/go"
)

func main() {
	ctx := context.Background()

	// In production, swap insecure.NewCredentials() for a TLS or mTLS credential. The server
	// authenticates no caller, so transport security alone protects it on the network.
	client, err := servicetemplate.NewClient(
		"service-template:8080",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	created, err := client.ItemCreate(ctx, &servicetemplate.ItemCreateRequest{
		Name:        "My Item",
		Description: "An optional description.",
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("created item %s", created.GetItem().GetId())
}
```

### JavaScript / TypeScript (REST)

The package is published to GitHub Packages, which requires a Personal Access Token with the `read:packages` scope even for public packages ([why](https://github.com/orgs/community/discussions/23386#discussioncomment-3240193)). Add to `.npmrc` (project root or `$HOME`):

```ini
@a-novel:registry=https://npm.pkg.github.com
@a-novel-kit:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=${YOUR_PERSONAL_ACCESS_TOKEN}
```

```bash
pnpm add @a-novel/service-template-rest
```

```typescript
import { TemplateApi, itemCreate, itemList } from "@a-novel/service-template-rest";

const api = new TemplateApi("http://service-template:8080");

const created = await itemCreate(api, "My Item", "An optional description.");
const items = await itemList(api, 10, 0);
```

## Running locally

The **standalone** images bundle a server with the migrations, which they apply on every boot. Use them for a throwaway instance without the dev toolchain, never for a deployment with several replicas.

```yaml
services:
  postgres-template:
    image: ghcr.io/a-novel/service-template/database:v0.0.0
    networks: [api]
    environment:
      POSTGRES_PASSWORD: postgres
      POSTGRES_USER: postgres
      POSTGRES_DB: postgres
      POSTGRES_HOST_AUTH_METHOD: scram-sha-256
      POSTGRES_INITDB_ARGS: --auth=scram-sha-256
    # Podman ignores the image's own check, so the dependents' wait needs this one.
    healthcheck:
      test: ["CMD", "pg_isready"]
      interval: 1s
      timeout: 5s
      retries: 10

  service-template:
    image: ghcr.io/a-novel/service-template/standalone-grpc:v0.0.0 # or standalone-rest
    ports: ["${GRPC_PORT}:8080"] # map ${REST_PORT} for the standalone-rest image
    depends_on:
      postgres-template: { condition: service_healthy }
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-template:5432/postgres?sslmode=disable"
    networks: [api]

networks:
  api:
```

## Contributing

Platform setup and the day-to-day commands live in the [developer guide](https://a-novel-kit.github.io/stack/). This repository's own concepts and local commands are in [CONTRIBUTING.md](./CONTRIBUTING.md).
