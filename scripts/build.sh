#!/bin/bash

set -e

# This script builds all the local dockerfiles under the ":local" tag.

podman build --format docker \
  -f ./builds/database.Dockerfile \
  -t ghcr.io/a-novel/service-narrative-engine/database:local .

podman build --format docker \
  -f ./builds/migrations.Dockerfile \
  -t ghcr.io/a-novel/service-narrative-engine/jobs/migrations:local .

podman build --format docker \
  -f ./builds/rest.Dockerfile \
  -t ghcr.io/a-novel/service-narrative-engine/rest:local .
podman build --format docker \
  -f ./builds/standalone.Dockerfile \
  -t ghcr.io/a-novel/service-narrative-engine/standalone:local .
