# Contributing to service-narrative-engine

Welcome to the narrative engine service for the A-Novel platform. This guide will help you understand the codebase, set
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

Create a `.envrc` file in the project root:

```bash
cp .envrc.template .envrc
```

Ask for an admin to replace placeholder values (prefixed with `SECRET_`).

Then, load the environment variables:

```bash
direnv allow .
# Alternatively, if you don't have direnv on your system
source .envrc
```

Finally, install the dependencies:

```bash
make install
```

### Common Commands

| Command           | Description                                           |
| ----------------- | ----------------------------------------------------- |
| `make run`        | Start all services locally                            |
| `make test`       | Run all tests                                         |
| `make test-short` | Run tests without AI calls (faster, less consumption) |
| `make lint`       | Run all linters                                       |
| `make format`     | Format all code                                       |
| `make build`      | Build Docker images locally                           |
| `make generate`   | Generate mocks and templates                          |

### Interacting with the Service

Once the service is running (`make run`), you can interact with it using `curl` or any HTTP client.

#### Health Checks

```bash
# Simple ping (is the server up?)
curl http://localhost:4021/ping

# Detailed health check (checks database, dependencies)
curl http://localhost:4021/healthcheck
```

#### Authentication

All endpoints (except health checks) require authentication. You need a valid access token from the
[authentication service](https://github.com/a-novel/service-authentication).

```bash
# Get an access token from the authentication service (see its CONTRIBUTING.md for details)
ACCESS_TOKEN="your_access_token_here"
```

#### Projects

Projects are the main containers for user narrative content.

```bash
# List your projects (supports pagination with limit/offset)
curl -X GET "http://localhost:4021/projects?limit=10&offset=0" \
  -H "Authorization: Bearer $ACCESS_TOKEN"

# Create a new project
curl -X PUT http://localhost:4021/projects \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{
    "lang": "en",
    "title": "My Story",
    "workflow": ["system:premise@v1.0.0", "system:character@v1.0.0"]
  }'

# Update a project (title and/or workflow)
curl -X PATCH http://localhost:4021/projects \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{
    "id": "<project-uuid>",
    "title": "Updated Title",
    "workflow": ["system:premise@v1.0.0", "system:character@v1.0.0", "system:plot@v1.0.0"]
  }'

# Delete a project
curl -X DELETE http://localhost:4021/projects \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{"id": "<project-uuid>"}'
```

#### Modules

Modules are predefined narrative templates with JSON schemas.

```bash
# Get a module by ID (format: namespace:id@vX.X.X[-preversion])
curl -X GET "http://localhost:4021/modules?module=system:character@v1.0.0" \
  -H "Authorization: Bearer $ACCESS_TOKEN"

# List available versions of a module
curl -X GET "http://localhost:4021/modules/versions?namespace=system&id=character" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

#### Schemas

Schemas store the actual narrative data for each module in a project.

```bash
# Get a schema (by module string or by schema id)
curl -X GET "http://localhost:4021/schemas?projectID=<project-uuid>&module=system:character@v1.0.0" \
  -H "Authorization: Bearer $ACCESS_TOKEN"

# List schema versions (returns history of edits for a module in a project)
curl -X GET "http://localhost:4021/schemas/versions?projectID=<project-uuid>&moduleNamespace=system&moduleID=character" \
  -H "Authorization: Bearer $ACCESS_TOKEN"

# Create a schema manually
curl -X PUT http://localhost:4021/schemas \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{
    "id": "<schema-uuid>",
    "projectID": "<project-uuid>",
    "module": "system:character@v1.0.0",
    "source": "USER",
    "data": {}
  }'

# Generate a schema using AI
curl -X PUT http://localhost:4021/schemas/generate \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{
    "projectID": "<project-uuid>",
    "module": "system:character@v1.0.0",
    "lang": "en"
  }'

# Rewrite/update a schema (creates a new version)
curl -X PATCH http://localhost:4021/schemas \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{
    "id": "<schema-uuid>",
    "data": {"name": "Updated character name"}
  }'
```

---

## Project Structure

- `/builds`: Build files for the containerized exports. Must be imported through the registry (see Readme).
- `/cmd`: Uncompiled commands of the service, used by the builds.
- `/internal`: this is where the core of the service lies. Compiled commands and containers use this code.
  It contains the hexagonal architecture layers.
  - `/config`: Configuration for the service, where editable values are loaded from various sources (yaml, env, etc.)
  - `/dao`: Data Access Objects, responsible for interacting with the data sources.
  - `/handlers`: Interface between the outside world and the application logic. Takes serialized requests, parse them
    for internal usage and serializes the response.
  - `/lib`: Shared utilities and helper functions used across the application.
  - `/models`: Defines external models used by the application (database migrations, templates, etc.)
  - `/services`: Core business logic of the application. Implements use cases and orchestrates interactions
    between different components.
- `/pkg`: Exported packages to interact with the service programmatically. Imported through dedicated package
  managers (see Readme)
- `/scripts`: Bash utils used during development and build.

---

## Project-Specific Guidelines

> This section contains patterns specific to this narrative engine service.

### Core Concepts

The narrative engine is built around three main entities:

| Entity  | Purpose                                                      |
| ------- | ------------------------------------------------------------ |
| Project | Container for a user's narrative work (story, novel, etc.)   |
| Module  | Predefined template defining the structure of narrative data |
| Schema  | User's actual data for a module within a project             |

### Module System

Modules define the structure for narrative elements using JSON Schema:

```
namespace:module-id@vX.X.X[-preversion]
```

**Examples:**

- `system:character@v1.0.0` - Character definition module
- `system:plot@v2.0.0-beta` - Plot structure module (prerelease)

**Module Components:**

- `id`: Unique identifier within the namespace
- `namespace`: Grouping (e.g., `system` for built-in modules)
- `version`: Semantic version
- `preversion`: Optional prerelease tag
- `description`: Human-readable description of the module
- `schema`: JSON Schema defining the data structure
- `ui`: UI hints for rendering forms
- `createdAt`: Timestamp of module creation

### AI Generation

The service integrates with OpenAI to generate narrative content:

1. User requests schema generation for a module
2. Service fetches context from existing schemas in the project
3. AI generates content following the module's JSON schema
4. Generated data is validated and stored

### Project Workflow

Each project has a `workflow` field defining the order of modules to complete:

```json
{
  "workflow": ["system:premise@v1.0.0", "system:character@v1.0.0", "system:plot@v1.0.0"]
}
```

This guides users through the narrative creation process step by step.

---

## Questions?

If you have questions or run into issues:

- Open an issue at https://github.com/a-novel/service-narrative-engine/issues
- Check existing issues for similar problems
- Include relevant logs and environment details
