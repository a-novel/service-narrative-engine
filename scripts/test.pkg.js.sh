#!/bin/bash

APP_NAME="service-template-integration-test-rest"
PODMAN_FILE="$PWD/builds/podman-compose.integration-test.rest.yaml"

# Ensure containers are properly shut down when the program exits abnormally.
int_handler()
{
    podman compose -p "${APP_NAME}" -f "${PODMAN_FILE}" down --volume
}
trap int_handler INT EXIT ERR

. "$PWD/scripts/setup-env.sh"

podman compose --podman-build-args='--format docker -q' -p "${APP_NAME}" -f "${PODMAN_FILE}" up -d --build

# Wait for services to be ready.
echo "Waiting for services to be ready..."

# Wait for JSON keys service.
MAX_RETRIES=30
RETRIES=0
until curl -s -o /dev/null -w "%{http_code}" "${REST_URL}/ping" | grep -q "200"; do
    RETRIES=$((RETRIES+1))
    [ $RETRIES -ge $MAX_RETRIES ] && exit 1
    echo "Waiting for JSON keys service on port ${REST_PORT}..."
    sleep 2
done
echo "JSON keys service is ready."

pnpm test || podman logs service-template-integration-test-rest_service-template_1

# Normal execution: containers are shut down.
podman compose -p "${APP_NAME}" -f "${PODMAN_FILE}" down --volume
