# Contributing to service-narrative-engine

Welcome to the Service Template for the A-Novel platform. This guide will help you understand the codebase, set
up your development environment, and contribute effectively.

Before reading this guide, if you haven't already, please check the
[generic contribution guidelines](https://github.com/a-novel/.github/blob/master/CONTRIBUTING.md) that are relevant
to your scope.

---

## Quick Start

### Prerequisites

The following must be installed on your system.

- [Go](https://go.dev/doc/install)
- [Node.js](https://nodejs.org/en/download)
  - [pnpm](https://pnpm.io/installation)
- [Podman](https://podman.io/docs/installation)
- (optional) [Direnv](https://direnv.net/)
- Make
  - `sudo apt-get install build-essential` (apt)
  - `sudo pacman -S make` (arch)
  - `brew install make` (macOS)
  - [Make for Windows](https://gnuwin32.sourceforge.net/packages/make.htm)

### Bootstrap

Install the dependencies:

```bash
go mod download
pnpm i --frozen-lockfile
```

### Secrets

The narrative engine calls an OpenAI-compatible API for text generation, so it
needs an API key to run end-to-end (the model-backed tests, `run`, and `ui`).
Provision it once with the CLI's secret manager — the value is stored encrypted
locally and injected into the service process only, never printed or committed
(the [onboarding guide](https://github.com/a-novel-kit/.github/blob/master/README.md)
explains how the manager works):

```bash
a-novel secrets set openai-key
```

Create the key at <https://platform.openai.com/api-keys>. The service maps it to
the `OPENAI_API_KEY` environment variable through its `.a-novel/secrets.yaml`, so
no further wiring is needed once the key is set; until then, model-backed runs
are skipped with a warning naming exactly what to set.

### Common Commands

| Command                                           | Description                            |
| ------------------------------------------------- | -------------------------------------- |
| `a-novel run start service-narrative-engine/rest` | Start the service locally (daemon)     |
| `a-novel test -y`                                 | Run all tests                          |
| `pnpm lint:go` / `pnpm lint:proto` / `pnpm lint`  | Run the linters (Go / proto / node)    |
| `pnpm format:go` / `pnpm format`                  | Format the code (Go / everything else) |
| `a-novel build -y`                                | Build Docker images locally            |
| `pnpm generate`                                   | Generate mocks and protobuf code       |

### Interacting with the Service

Once the service is running (`a-novel run start service-narrative-engine/rest`), you can interact with it using:

- `curl` or any HTTP client (REST API).
- `grpcurl` or any gRPC client (gRPC API).

#### Health Checks

```bash
# REST: Simple ping (is the server up?)
curl http://localhost:${SERVICE_NARRATIVE_ENGINE_REST_PORT}/ping

# REST: Detailed health check (checks database, dependencies)
curl http://localhost:${SERVICE_NARRATIVE_ENGINE_REST_PORT}/healthcheck

# gRPC: Simple ping (is the server up?)
grpcurl -plaintext localhost:${SERVICE_NARRATIVE_ENGINE_GRPC_PORT} grpc.health.v1.Health/Check

# gRPC: Check the status of all services.
grpcurl -plaintext localhost:${SERVICE_NARRATIVE_ENGINE_GRPC_PORT} StatusService/Status
```

#### Item Operations

List items:

```bash
# REST
curl "http://localhost:${SERVICE_NARRATIVE_ENGINE_REST_PORT}/items?limit=10&offset=0"

# gRPC
grpcurl -plaintext -d '{"limit": 10, "offset": 0}' "localhost:${SERVICE_NARRATIVE_ENGINE_GRPC_PORT}" ItemListService/ItemList
```

Get a specific item:

```bash
# REST
curl "http://localhost:${SERVICE_NARRATIVE_ENGINE_REST_PORT}/item?id=<item-uuid>"

# gRPC
grpcurl -plaintext -d '{"id": "<item-uuid>"}' "localhost:${SERVICE_NARRATIVE_ENGINE_GRPC_PORT}" ItemGetService/ItemGet
```

Create an item:

```bash
# REST
curl -X POST "http://localhost:${SERVICE_NARRATIVE_ENGINE_REST_PORT}/items" \
  -H "Content-Type: application/json" \
  -d '{"name": "My Item", "description": "An optional description."}'

# gRPC
grpcurl -plaintext -d '{"name": "My Item", "description": "An optional description."}' \
  "localhost:${SERVICE_NARRATIVE_ENGINE_GRPC_PORT}" ItemCreateService/ItemCreate
```

Update an item:

```bash
# REST
curl -X PUT "http://localhost:${SERVICE_NARRATIVE_ENGINE_REST_PORT}/item" \
  -H "Content-Type: application/json" \
  -d '{"id": "<item-uuid>", "name": "Updated Name", "description": "Updated description."}'

# gRPC
grpcurl -plaintext -d '{"id": "<item-uuid>", "name": "Updated Name"}' \
  "localhost:${SERVICE_NARRATIVE_ENGINE_GRPC_PORT}" ItemUpdateService/ItemUpdate
```

Delete an item:

```bash
# REST
curl -X DELETE "http://localhost:${SERVICE_NARRATIVE_ENGINE_REST_PORT}/item?id=<item-uuid>"

# gRPC
grpcurl -plaintext -d '{"id": "<item-uuid>"}' "localhost:${SERVICE_NARRATIVE_ENGINE_GRPC_PORT}" ItemDeleteService/ItemDelete
```

---

## Project-Specific Guidelines

> This section contains patterns specific to this service template.

### Item Resource

The `item` is the placeholder resource in this template. It represents a simple named entity with an optional
description. When adapting this template for a real service, replace the `item` resource with your actual domain
model across all layers (DAO, services, handlers, proto, migrations).

### Layer Architecture

The codebase is organized into three layers:

| Layer   | Package              | Responsibility                              |
| ------- | -------------------- | ------------------------------------------- |
| DAO     | `internal/dao/`      | Raw database queries using bun ORM          |
| Service | `internal/services/` | Validation, business logic, UUID generation |
| Handler | `internal/handlers/` | gRPC and REST request/response mapping      |

### gRPC Services

| Service             | Purpose                      |
| ------------------- | ---------------------------- |
| `StatusService`     | Health and status checks     |
| `ItemCreateService` | Create a new item            |
| `ItemGetService`    | Retrieve a single item by ID |
| `ItemListService`   | List items (paginated)       |
| `ItemUpdateService` | Update an existing item      |
| `ItemDeleteService` | Delete an item by ID         |

### REST API

The REST API serves item data over HTTP. It is documented via an OpenAPI spec:

- `openapi.yaml` — machine-readable spec
- `openapi.html` — interactive Scalar API Reference viewer (open in a browser)

The published API reference is hosted at [GitHub Pages](https://a-novel.github.io/service-narrative-engine).

### JavaScript Client Package

Frontend or Node.js consumers can use `@a-novel/service-narrative-engine-rest` (`pkg/js/rest/`) to call the REST API:

```typescript
import {
  TemplateApi,
  itemCreate,
  itemDelete,
  itemGet,
  itemList,
  itemUpdate,
} from "@a-novel/service-narrative-engine-rest";

const api = new TemplateApi("http://localhost:8080");

// Check service health.
await api.ping();

// Manage items.
const created = await itemCreate(api, "My Item", "An optional description.");
const items = await itemList(api, 10, 0);
const item = await itemGet(api, "<item-id>");
const updated = await itemUpdate(api, "<item-id>", "Updated Name");
const deleted = await itemDelete(api, "<item-id>");
```

Integration tests for the JS client live in `pkg/js/test/rest/`. Run them locally with:

```bash
a-novel test --type=pnpm -y
```

### Go Client Package

Other services integrate with the service via the `pkg/` package:

```go
import pkg "github.com/a-novel/service-narrative-engine/pkg"

// Create client
client, err := pkg.NewClient("<grpc-address>")
defer client.Close()

// Create an item
res, err := client.ItemCreate(ctx, &pkg.ItemCreateRequest{
    Name:        "My Item",
    Description: "An optional description.",
})

// List items
list, err := client.ItemList(ctx, &pkg.ItemListRequest{Limit: 10})
```

---

## Questions?

If you have questions or run into issues:

- Open an issue at https://github.com/a-novel/service-narrative-engine/issues
- Check existing issues for similar problems
- Include relevant logs and environment details
