#!/bin/bash

APP_NAME="service-narrative-engine-integration-test"
PODMAN_FILE="$PWD/builds/podman-compose.integration-test.yaml"

# Ensure containers are properly shut down when the program exits abnormally.
int_handler()
{
    podman compose -p "${APP_NAME}" -f "${PODMAN_FILE}" down --volume
}
trap int_handler INT

podman compose --podman-build-args='--format docker -q' -p "${APP_NAME}" -f "${PODMAN_FILE}" up -d --build

# Wait for services to be ready.
echo "Waiting for services to be ready..."

# Wait for auth service (port 4011).
until curl -s -o /dev/null -w "%{http_code}" "http://localhost:${AUTH_API_PORT:-4011}/ping" | grep -q "200"; do
    echo "Waiting for auth service..."
    sleep 2
done
echo "Auth service is ready."

# Wait for narrative engine service (port 4021).
until curl -s -o /dev/null -w "%{http_code}" "http://localhost:${API_PORT:-4021}/ping" | grep -q "200"; do
    echo "Waiting for narrative engine service..."
    sleep 2
done
echo "Narrative engine service is ready."

pnpm test

# Normal execution: containers are shut down.
podman compose -p "${APP_NAME}" -f "${PODMAN_FILE}" down --volume
