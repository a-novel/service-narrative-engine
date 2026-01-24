# Narrative engine service

[![X (formerly Twitter) Follow](https://img.shields.io/twitter/follow/agorastoryverse)](https://twitter.com/agorastoryverse)
[![Discord](https://img.shields.io/discord/1315240114691248138?logo=discord)](https://discord.gg/rp4Qr8cA)

<hr />

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/a-novel/service-narrative-engine)
![GitHub repo file or directory count](https://img.shields.io/github/directory-file-count/a-novel/service-narrative-engine)
![GitHub code size in bytes](https://img.shields.io/github/languages/code-size/a-novel/service-narrative-engine)

![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/a-novel/service-narrative-engine/main.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/a-novel/service-narrative-engine)](https://goreportcard.com/report/github.com/a-novel/service-narrative-engine)
[![codecov](https://codecov.io/gh/a-novel/service-narrative-engine/graph/badge.svg?token=pjvCjxURPS)](https://codecov.io/gh/a-novel/service-narrative-engine)

![Coverage graph](https://codecov.io/gh/a-novel/service-narrative-engine/graphs/sunburst.svg?token=pjvCjxURPS)

## Usage

### Docker

Run the service as a containerized application (the below examples use docker-compose syntax).

```yaml
services:
  postgres-narrative-engine:
    image: ghcr.io/a-novel/service-narrative-engine/database:v0.0.1
    networks:
      - api
    environment:
      POSTGRES_PASSWORD: postgres
      POSTGRES_USER: postgres
      POSTGRES_DB: postgres
      POSTGRES_HOST_AUTH_METHOD: scram-sha-256
      POSTGRES_INITDB_ARGS: --auth=scram-sha-256
    volumes:
      - narrative-engine-postgres-data:/var/lib/postgresql/

  service-narrative-engine:
    image: ghcr.io/a-novel/service-narrative-engine/standalone:v0.0.1
    ports:
      - "4021:8080"
    depends_on:
      postgres-narrative-engine:
        condition: service_healthy
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-narrative-engine:5432/postgres?sslmode=disable"
      SERVICE_JSON_KEYS_PORT: # Port where service-json-keys is running
      SERVICE_JSON_KEYS_HOST: # Host name of service-json-keys instance
      OPENAI_API_KEY: # Your OpenAI API key
      OPENAI_BASE_URL: # OpenAI API base URL (optional)
      OPENAI_MODEL: # OpenAI model to use for generation
    networks:
      - api

networks:
  api:

volumes:
  narrative-engine-postgres-data:
```

Note the standalone image is an all-in-one initializer for the application; however, it runs heavy operations such
as migrations on every launch. Thus, while it comes in handy for local development, it is NOT RECOMMENDED for
production deployments. Instead, consider using the separate, optimized images for that purpose.

```yaml
services:
  postgres-narrative-engine:
    image: ghcr.io/a-novel/service-narrative-engine/database:v0.0.1
    networks:
      - api
    environment:
      POSTGRES_PASSWORD: postgres
      POSTGRES_USER: postgres
      POSTGRES_DB: postgres
      POSTGRES_HOST_AUTH_METHOD: scram-sha-256
      POSTGRES_INITDB_ARGS: --auth=scram-sha-256
    volumes:
      - narrative-engine-postgres-data:/var/lib/postgresql/

  migrations-narrative-engine:
    image: ghcr.io/a-novel/service-narrative-engine/migrations:v0.0.1
    depends_on:
      postgres-narrative-engine:
        condition: service_healthy
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-narrative-engine:5432/postgres?sslmode=disable"
    networks:
      - api

  # Optional job, used to inject base data (system modules) into a freshly initialized database.
  init-narrative-engine:
    image: ghcr.io/a-novel/service-narrative-engine/init:v0.0.1
    depends_on:
      postgres-narrative-engine:
        condition: service_healthy
      migrations-narrative-engine:
        condition: service_completed_successfully
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-narrative-engine:5432/postgres?sslmode=disable"
    networks:
      - api

  service-narrative-engine:
    image: ghcr.io/a-novel/service-narrative-engine/rest:v0.0.1
    ports:
      - "4021:8080"
    depends_on:
      postgres-narrative-engine:
        condition: service_healthy
      migrations-narrative-engine:
        condition: service_completed_successfully
      init-narrative-engine:
        condition: service_completed_successfully
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-narrative-engine:5432/postgres?sslmode=disable"
      SERVICE_JSON_KEYS_PORT: # Port where service-json-keys is running
      SERVICE_JSON_KEYS_HOST: # Host name of service-json-keys instance
      OPENAI_API_KEY: # Your OpenAI API key
      OPENAI_BASE_URL: # OpenAI API base URL (optional)
      OPENAI_MODEL: # OpenAI model to use for generation
    networks:
      - api

networks:
  api:

volumes:
  narrative-engine-postgres-data:
```

Above are the minimal required configurations to run the service locally. Configuration is done through environment
variables. Below is a list of available configurations:

**Required variables**

| Name                   | Description                                                          | Images                                              |
| ---------------------- | -------------------------------------------------------------------- | --------------------------------------------------- |
| POSTGRES_DSN           | The Postgres Data Source Name (DSN) used to connect to the database. | `standalone`<br/>`rest`<br/>`init`<br/>`migrations` |
| SERVICE_JSON_KEYS_PORT | Port where service-json-keys is running                              | `standalone`<br/>`rest`                             |
| SERVICE_JSON_KEYS_HOST | Host name of a running service-json-keys instance (without protocol) | `standalone`<br/>`rest`                             |
| OPENAI_API_KEY         | Your OpenAI API key for content generation                           | `standalone`<br/>`rest`                             |
| OPENAI_MODEL           | OpenAI model to use for generation                                   | `standalone`<br/>`rest`                             |

This service requires a running instance of the [JSON Keys service](https://github.com/a-novel/service-json-keys). Note
that the narrative engine and json keys service share sensitive data, they should communicate over a secure, unexposed
network.

**OpenAI Configuration**

| Name            | Description                 | Images                  |
| --------------- | --------------------------- | ----------------------- |
| OPENAI_BASE_URL | Base URL for the OpenAI API | `standalone`<br/>`rest` |

**Rest API**

While you should not need to change these values in most cases, the following variables allow you to
customize the API behavior.

| Name                       | Description                                 | Default value    | Images                  |
| -------------------------- | ------------------------------------------- | ---------------- | ----------------------- |
| API_PORT                   | Port on which the API listens               | `8080`           | `standalone`<br/>`rest` |
| API_MAX_REQUEST_SIZE       | Maximum size of incoming requests in bytes  | `2097152` (2MiB) | `standalone`<br/>`rest` |
| API_TIMEOUT_READ           | Timeout for read operations                 | `15s`            | `standalone`<br/>`rest` |
| API_TIMEOUT_READ_HEADER    | Timeout for header reading operations       | `3s`             | `standalone`<br/>`rest` |
| API_TIMEOUT_WRITE          | Timeout for write operations                | `30s`            | `standalone`<br/>`rest` |
| API_TIMEOUT_IDLE           | Idle timeout                                | `60s`            | `standalone`<br/>`rest` |
| API_TIMEOUT_REQUEST        | Timeout for api requests                    | `60s`            | `standalone`<br/>`rest` |
| API_CORS_ALLOWED_ORIGINS   | CORS allowed origins (allow all by default) | `*`              | `standalone`<br/>`rest` |
| API_CORS_ALLOWED_HEADERS   | CORS allowed headers (allow all by default) | `*`              | `standalone`<br/>`rest` |
| API_CORS_ALLOW_CREDENTIALS | CORS allow credentials                      | `false`          | `standalone`<br/>`rest` |
| API_CORS_MAX_AGE           | CORS max age                                | `3600`           | `standalone`<br/>`rest` |

**Logs & Tracing**

For now, OTEL is only provided using 2 exporters: stdout and Google Cloud. Other integrations may come
in the future.

| Name              | Description                                                                             | Default value              | Images                             |
| ----------------- | --------------------------------------------------------------------------------------- | -------------------------- | ---------------------------------- |
| OTEL              | Activate OTEL tracing (use options below to switch between exporters)                   | `false`                    | `standalone`<br/>`rest`<br/>`init` |
| GCLOUD_PROJECT_ID | Google Cloud project id for the OTEL exporter. Switch to Google Cloud exporter when set |                            | `standalone`<br/>`rest`<br/>`init` |
| APP_NAME          | Application name to be used in traces                                                   | `service-narrative-engine` | `standalone`<br/>`rest`<br/>`init` |

**Setup**

The below variables allow you to configure the service initialization. Note that `VERSION` is automatically
extracted from `package.json` in both the `standalone` and `init` images.

| Name     | Description                                            | Images                  |
| -------- | ------------------------------------------------------ | ----------------------- |
| DEV_MODE | Enable development mode features (e.g., preversioning) | `standalone`<br/>`init` |

### Javascript (npm)

To interact with a running instance of the narrative engine service, you can use the integrated package.

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
pnpm add @a-novel/service-narrative-engine-rest
```

To use it, you must create a `NarrativeEngineApi` instance. A single instance can be shared across
your client.

```typescript
import { NarrativeEngineApi } from "@a-novel/service-narrative-engine-rest";

export const narrativeEngineApi = new NarrativeEngineApi("<base_api_url>");

// (optional) check the status of the api connection.
await narrativeEngineApi.ping();
await narrativeEngineApi.health();
```

You can then call methods from the package using this api instance. Each method comes with
[zod](https://github.com/colinhacks/zod) types so you can validate requests easily.

Responses are validated by default.

```typescript
import {
  ModuleListVersionsRequestSchema,
  // Module types and methods
  ModuleSchema,
  ModuleSelectRequestSchema,
  ModuleVersionEntrySchema,
  ProjectDeleteRequestSchema,
  ProjectInitRequestSchema,
  ProjectListRequestSchema,
  // Project types and methods
  ProjectSchema,
  ProjectUpdateRequestSchema,
  SchemaCreateRequestSchema,
  SchemaGenerateRequestSchema,
  SchemaListVersionsRequestSchema,
  SchemaRewriteRequestSchema,
  // Schema types and methods
  SchemaSchema,
  SchemaSelectRequestSchema,
  moduleListVersions,
  moduleSelect,
  projectDelete,
  projectInit,
  projectList,
  projectUpdate,
  schemaCreate,
  schemaGenerate,
  schemaListVersions,
  schemaRewrite,
  schemaSelect,
} from "@a-novel/service-narrative-engine-rest";
```

### Go services integration

To integrate with this service from your Go applications, you'll need to set up authentication using the
[JSON Keys service](https://github.com/a-novel/service-json-keys) and
[Authentication service](https://github.com/a-novel/service-authentication). See those repositories for
detailed integration instructions.

The narrative engine service itself is consumed via its REST API. Use the JavaScript package above for
type-safe client integration or make HTTP requests directly to the API endpoints.
