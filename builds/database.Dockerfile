# A postgres image with the extensions the service needs pre-loaded at build time.
#
# It does not run the service's schema migrations; run the migrations target separately.
FROM docker.io/library/postgres:18.4

ARG DEBIAN_FRONTEND=noninteractive

# ======================================================================================================================
# Prepare extension scripts.
# ======================================================================================================================
# Entrypoint that starts postgres with the extension settings applied.
COPY ./builds/database.entrypoint.sh /usr/local/bin/database.entrypoint.sh
RUN chmod +x /usr/local/bin/database.entrypoint.sh

# Runs once on an empty data directory, to create the extensions.
COPY ./builds/database.sql /docker-entrypoint-initdb.d/init.sql

# ======================================================================================================================
# Finish setup.
# ======================================================================================================================
EXPOSE 5432

# Postgres does not provide a healthcheck by default.
HEALTHCHECK --interval=1s --timeout=5s --retries=10 --start-period=1s \
  CMD pg_isready || exit 1

ENTRYPOINT ["/usr/local/bin/database.entrypoint.sh"]

# Restore the original command from the base image.
CMD ["postgres"]
