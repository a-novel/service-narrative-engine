#!/bin/bash

set -e

# This script builds all the dockerfiles under the ":local" tag.

podman build --format docker \
  -f ./builds/database.Dockerfile \
  -t ghcr.io/a-novel/service-template/database:local .

podman build --format docker \
  -f ./builds/migrations.Dockerfile \
  -t ghcr.io/a-novel/service-template/jobs/migrations:local .

podman build --format docker \
  -f ./builds/grpc.Dockerfile \
  -t ghcr.io/a-novel/service-template/grpc:local .
podman build --format docker \
  -f ./builds/standalone.grpc.Dockerfile \
  -t ghcr.io/a-novel/service-template/standalone-grpc:local .

podman build --format docker \
  -f ./builds/rest.Dockerfile \
  -t ghcr.io/a-novel/service-template/rest:local .
podman build --format docker \
  -f ./builds/standalone.rest.Dockerfile \
  -t ghcr.io/a-novel/service-template/standalone-rest:local .
