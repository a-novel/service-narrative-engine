#!/bin/bash
set -e

# Default value for POSTGRES_DB if not set
POSTGRES_DB=${POSTGRES_DB:-postgres}

# Execute the original entrypoint script, with extra arguments.
exec /usr/local/bin/docker-entrypoint.sh "$@" -c "cron.database_name=${POSTGRES_DB}"
