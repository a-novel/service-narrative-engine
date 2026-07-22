#!/bin/bash
# Wraps the postgres image's own entrypoint, pointing pg_cron at the database the
# container serves.
set -e

POSTGRES_DB=${POSTGRES_DB:-postgres}

exec /usr/local/bin/docker-entrypoint.sh "$@" -c "cron.database_name=${POSTGRES_DB}"
