# Service Template

[![X (formerly Twitter) Follow](https://img.shields.io/twitter/follow/agorastoryverse)](https://twitter.com/agorastoryverse)
[![Discord](https://img.shields.io/discord/1315240114691248138?logo=discord)](https://discord.gg/rp4Qr8cA)

<hr />

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/a-novel/service-template)
![GitHub repo file or directory count](https://img.shields.io/github/directory-file-count/a-novel/service-template)
![GitHub code size in bytes](https://img.shields.io/github/languages/code-size/a-novel/service-template)

![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/a-novel/service-template/main.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/a-novel/service-template)](https://goreportcard.com/report/github.com/a-novel/service-template)
[![codecov](https://codecov.io/gh/a-novel/service-template/graph/badge.svg?token=XK5D0l728E)](https://codecov.io/gh/a-novel/service-template)

![Coverage graph](https://codecov.io/gh/a-novel/service-template/graphs/sunburst.svg?token=XK5D0l728E)

## Using this template

This repository is a starting point for a new Agora backend service, not a service you deploy as
is. It ships one dummy resource — `item`, a trivial CRUD entity — wired end to end (DB → DAO →
service → REST + gRPC handlers → Go/JS clients → OpenAPI) so every layer has a working example to
copy. Replace it with your own resource, rename the module, and you have a working service.

### 1. Rename the module and the service identity

| What                   | Where                                                                                                                  | From → to                                                                   |
| ---------------------- | ---------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------- |
| Go module path         | `go.mod` (`module …`) + every import in `internal/`, `cmd/`, `pkg/go/`, `*_test.go`                                    | `github.com/a-novel/service-template` → `github.com/a-novel/<your-service>` |
| Module path in tooling | `.mockery.yaml`, `buf.gen.yaml`                                                                                        | same rename                                                                 |
| Root generate file     | `generate.go` (`package …`)                                                                                            | the placeholder package name → `package <yourservice>`                      |
| Env-var prefix         | `internal/config/env/env.go` (`os.Getenv("SERVICE_TEMPLATE_ENV_PREFIX")`)                                              | `SERVICE_TEMPLATE_ENV_PREFIX` → `<YOUR_SERVICE>_ENV_PREFIX`                 |
| App name default       | `internal/config/env/env.go` (`AppNameDefault`)                                                                        | `"service-template"` → `"<your-service>"`                                   |
| Repo references        | `README.md` (badges, image names, doc links), CI workflows (`image_name:`), `package.json`, `pkg/js/rest/package.json` | `service-template` → `<your-service>`                                       |

A `go mod edit -module github.com/a-novel/<your-service>` followed by a project-wide find/replace
of `service-template` → `<your-service>` covers most of it (the import paths use the
`github.com/a-novel/service-template/internal/...` form, so the same replace catches them);
then `make format` and `make generate`.

### 2. Replace the `item` resource

Swap `item` for your own resource (call it `widget`, say) — touch each of these, renaming
`item`/`Item` → `widget`/`Widget` and adjusting fields:

- **Migration** — `internal/models/migrations/20250306000000_items_table.{up,down}.sql` (and `make migrations` / `date '+%Y%m%d%H%M%S'` for a fresh timestamp if you'd rather start over).
- **DAO** — `internal/dao/pg.item.go` (the bun model), `pg.itemCreate.go` / `.sql` and the `Get` / `List` / `Update` / `Delete` siblings, plus the `*_test.go` files.
- **Services** — `internal/services/item.go`, `itemCreate.go` … `itemDelete.go`, the `*_test.go` files, and `internal/services/validate.go` if your fields need different custom validators (it currently only registers `notblank`).
- **Handlers** — REST: `internal/handlers/http.item.go`, `http.itemCreatePublic.go` … `http.itemDeletePublic.go`; gRPC: `internal/handlers/grpc.itemCreate.go` … `grpc.itemDelete.go`; plus the `*_test.go` files.
- **Proto** — `internal/models/proto/item*.proto` (then `make generate` to refresh `internal/handlers/protogen/`).
- **API surface** — `openapi.yaml` (+ regenerate `openapi.html`), `pkg/go/client.go`, `pkg/js/rest/src/item.ts` and `pkg/js/rest/src/index.ts`, and the integration tests under `pkg/js/test/`.
- **Wiring** — the constructor calls and route registrations in `cmd/rest/main.go` and `cmd/grpc/main.go`.

### 3. Keep the scaffolding

These are not dummy code — leave them in place (adjusting only as your service needs):

- `internal/config/` (env loading, presets), the `cmd/*/main.go` startup shape (config → otel → DB
  context → DAOs → services → handlers → routes → graceful shutdown), `internal/handlers/http.ping.go`
  / `http.health.go` / `http.decoder.go` / `grpc.status.go`, `internal/models/migrations/migrations.go`.
- `Makefile`, `builds/`, `scripts/`, `.github/workflows/`, the `*.mod` tool-dep files, `renovate.json`,
  `buf.*`, the prettier/pnpm config — all org-standard, shared with the other services.
- `LICENSE`, `SECURITY.md`, `CODE_OF_CONDUCT.md`, `CONTRIBUTING.md` — update only the bits that name
  the service.

### 4. Verify

`make generate` → `make format` → `make lint` → `make test`. Then refresh `openapi.html`, update
this README (delete this section, fix the badges/links/Docker image names), and you have a service.

Conventions for writing the Go (layering, naming, error handling, telemetry, tests) are governed by
the `write-go` / `write-go-service` / `write-go-tests` skills in the agora `stack` repo; `write-sql`,
`write-proto`, `write-openapi`, `write-dockerfiles` cover the rest.

## Usage

### Docker

Run the service as a containerized application (the below examples use docker-compose syntax).

#### gRPC

> Set the SERVICE_TEMPLATE_GRPC_PORT env variable to whatever port you want to use for the service.

```yaml
services:
  postgres-template:
    image: ghcr.io/a-novel/service-template/database:v0.0.0
    networks:
      - api
    environment:
      POSTGRES_PASSWORD: postgres
      POSTGRES_USER: postgres
      POSTGRES_DB: postgres
      POSTGRES_HOST_AUTH_METHOD: scram-sha-256
      POSTGRES_INITDB_ARGS: --auth=scram-sha-256
    volumes:
      - template-postgres-data:/var/lib/postgresql/

  service-template:
    image: ghcr.io/a-novel/service-template/standalone-grpc:v0.0.0
    ports:
      - "${SERVICE_TEMPLATE_GRPC_PORT}:8080"
    depends_on:
      postgres-template:
        condition: service_healthy
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-template:5432/postgres?sslmode=disable"
    networks:
      - api

networks:
  api:

volumes:
  template-postgres-data:
```

Note the standalone image is an all-in-one initializer for the application; however, it runs heavy operations such
as migrations on every launch. Thus, while it comes in handy for local development, it is NOT RECOMMENDED for
production deployments. Instead, consider using the separate, optimized images for that purpose.

```yaml
services:
  postgres-template:
    image: ghcr.io/a-novel/service-template/database:v0.0.0
    networks:
      - api
    environment:
      POSTGRES_PASSWORD: postgres
      POSTGRES_USER: postgres
      POSTGRES_DB: postgres
      POSTGRES_HOST_AUTH_METHOD: scram-sha-256
      POSTGRES_INITDB_ARGS: --auth=scram-sha-256
    volumes:
      - template-postgres-data:/var/lib/postgresql/

  migrations-template:
    image: ghcr.io/a-novel/service-template/jobs/migrations:v0.0.0
    depends_on:
      postgres-template:
        condition: service_healthy
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-template:5432/postgres?sslmode=disable"
    networks:
      - api

  service-template:
    image: ghcr.io/a-novel/service-template/grpc:v0.0.0
    ports:
      - "${SERVICE_TEMPLATE_GRPC_PORT}:8080"
    depends_on:
      postgres-template:
        condition: service_healthy
      migrations-template:
        condition: service_completed_successfully
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-template:5432/postgres?sslmode=disable"
    networks:
      - api

networks:
  api:

volumes:
  template-postgres-data:
```

#### REST

> Set the SERVICE_TEMPLATE_REST_PORT env variable to whatever port you want to use for the service.

```yaml
services:
  postgres-template:
    image: ghcr.io/a-novel/service-template/database:v0.0.0
    networks:
      - api
    environment:
      POSTGRES_PASSWORD: postgres
      POSTGRES_USER: postgres
      POSTGRES_DB: postgres
      POSTGRES_HOST_AUTH_METHOD: scram-sha-256
      POSTGRES_INITDB_ARGS: --auth=scram-sha-256
    volumes:
      - template-postgres-data:/var/lib/postgresql/

  service-template:
    image: ghcr.io/a-novel/service-template/standalone-rest:v0.0.0
    ports:
      - "${SERVICE_TEMPLATE_REST_PORT}:8080"
    depends_on:
      postgres-template:
        condition: service_healthy
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-template:5432/postgres?sslmode=disable"
    networks:
      - api

networks:
  api:

volumes:
  template-postgres-data:
```

Note the standalone image is an all-in-one initializer for the application; however, it runs heavy operations such
as migrations on every launch. Thus, while it comes in handy for local development, it is NOT RECOMMENDED for
production deployments. Instead, consider using the separate, optimized images for that purpose.

```yaml
services:
  postgres-template:
    image: ghcr.io/a-novel/service-template/database:v0.0.0
    networks:
      - api
    environment:
      POSTGRES_PASSWORD: postgres
      POSTGRES_USER: postgres
      POSTGRES_DB: postgres
      POSTGRES_HOST_AUTH_METHOD: scram-sha-256
      POSTGRES_INITDB_ARGS: --auth=scram-sha-256
    volumes:
      - template-postgres-data:/var/lib/postgresql/

  migrations-template:
    image: ghcr.io/a-novel/service-template/jobs/migrations:v0.0.0
    depends_on:
      postgres-template:
        condition: service_healthy
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-template:5432/postgres?sslmode=disable"
    networks:
      - api

  service-template:
    image: ghcr.io/a-novel/service-template/rest:v0.0.0
    ports:
      - "${SERVICE_TEMPLATE_REST_PORT}:8080"
    depends_on:
      postgres-template:
        condition: service_healthy
      migrations-template:
        condition: service_completed_successfully
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-template:5432/postgres?sslmode=disable"
    networks:
      - api

networks:
  api:

volumes:
  template-postgres-data:
```

Above are the minimal required configuration to run the service locally. Configuration is done through environment
variables. Below is a list of available configurations:

**Required variables**

| Name         | Description                                                          | Images                                                                         |
| ------------ | -------------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| POSTGRES_DSN | The Postgres Data Source Name (DSN) used to connect to the database. | `standalone-grpc`<br/>`standalone-rest`<br/>`grpc`<br/>`rest`<br/>`migrations` |

**REST API**

While you should not need to change these values in most cases, the following variables allow you to
customize the REST API behavior.

| Name                        | Description                                 | Default value    | Images                       |
| --------------------------- | ------------------------------------------- | ---------------- | ---------------------------- |
| REST_MAX_REQUEST_SIZE       | Maximum size of incoming requests in bytes  | `2097152` (2MiB) | `standalone-rest`<br/>`rest` |
| REST_TIMEOUT_READ           | Timeout for read operations                 | `15s`            | `standalone-rest`<br/>`rest` |
| REST_TIMEOUT_READ_HEADER    | Timeout for header reading operations       | `3s`             | `standalone-rest`<br/>`rest` |
| REST_TIMEOUT_WRITE          | Timeout for write operations                | `30s`            | `standalone-rest`<br/>`rest` |
| REST_TIMEOUT_IDLE           | Idle timeout                                | `60s`            | `standalone-rest`<br/>`rest` |
| REST_TIMEOUT_REQUEST        | Timeout for api requests                    | `60s`            | `standalone-rest`<br/>`rest` |
| REST_CORS_ALLOWED_ORIGINS   | CORS allowed origins (allow all by default) | `*`              | `standalone-rest`<br/>`rest` |
| REST_CORS_ALLOWED_HEADERS   | CORS allowed headers (allow all by default) | `*`              | `standalone-rest`<br/>`rest` |
| REST_CORS_ALLOW_CREDENTIALS | CORS allow credentials                      | `false`          | `standalone-rest`<br/>`rest` |
| REST_CORS_MAX_AGE           | CORS max age                                | `3600`           | `standalone-rest`<br/>`rest` |

**Logs & Tracing**

For now, OTEL is only provided using 2 exporters: stdout and Google Cloud. Other integrations may come
in the future.

| Name              | Description                                                                             | Default value      | Images                                                        |
| ----------------- | --------------------------------------------------------------------------------------- | ------------------ | ------------------------------------------------------------- |
| OTEL              | Activate OTEL tracing (use options below to switch between exporters)                   | `false`            | `standalone-grpc`<br/>`standalone-rest`<br/>`grpc`<br/>`rest` |
| GCLOUD_PROJECT_ID | Google Cloud project id for the OTEL exporter. Switch to Google Cloud exporter when set |                    | `standalone-grpc`<br/>`standalone-rest`<br/>`grpc`<br/>`rest` |
| APP_NAME          | Application name to be used in traces                                                   | `service-template` | `standalone-grpc`<br/>`standalone-rest`<br/>`grpc`<br/>`rest` |

### Javascript (npm)

To interact with a running REST instance of the service, you can use the integrated package.

> ⚠️ **Warning**: Even though the package is public, GitHub registry requires you to have a Personal Access Token
> with `repo` and `read:packages` scopes to pull it in your project. See
> [this issue](https://github.com/orgs/community/discussions/23386#discussioncomment-3240193) for more information.

Make sure you have a `.npmrc` with the following content (in your project or in your home directory):

```ini
@a-novel:registry=https://npm.pkg.github.com
@a-novel-kit:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=${YOUR_PERSONAL_ACCESS_TOKEN}
```

Then, install the package using pnpm:

```bash
# pnpm config set auto-install-peers true
#  Or
# pnpm config set auto-install-peers true --location project
pnpm add @a-novel/service-template-rest
```

To use it, create a `TemplateApi` instance. A single instance can be shared across your client.

```typescript
import { TemplateApi, itemCreate, itemDelete, itemGet, itemList, itemUpdate } from "@a-novel/service-template-rest";

export const templateApi = new TemplateApi("<base_api_url>");

// (optional) check the status of the api connection.
await templateApi.ping();
await templateApi.health();
```

Manage items:

```typescript
// Create a new item.
const created = await itemCreate(templateApi, "My Item", "An optional description.");

// List items (paginated).
const items = await itemList(templateApi, 10, 0);

// Get a specific item by ID.
const item = await itemGet(templateApi, "<item-id>");

// Update an item.
const updated = await itemUpdate(templateApi, "<item-id>", "Updated Name", "Updated description.");

// Delete an item.
const deleted = await itemDelete(templateApi, "<item-id>");
```

The API reference is available at [GitHub Pages](https://a-novel.github.io/service-template).

### Go module

You can integrate the service capabilities directly into your Go services by using the provided
Go module. It requires a connection to a running gRPC instance of this service.

```bash
go get -u github.com/a-novel/service-template
```

```go
package main

import (
  "context"
  "fmt"

  "google.golang.org/grpc"
  "google.golang.org/grpc/credentials/insecure"

  pkg "github.com/a-novel/service-template/pkg"
)

func main() {
  ctx := context.Background()

  client, _ := pkg.NewClient(
    "<service-template-grpc-url>",
    grpc.WithTransportCredentials(insecure.NewCredentials()),
  )
  defer client.Close()

  // Create an item.
  res, _ := client.ItemCreate(ctx, &pkg.ItemCreateRequest{
    Name:        "My Item",
    Description: "An optional description.",
  })

  fmt.Println(res.GetItem().GetId())

  // List items.
  list, _ := client.ItemList(ctx, &pkg.ItemListRequest{Limit: 10})
  for _, item := range list.GetItems() {
    fmt.Println(item.GetName())
  }
}
```
