# Exposes the service as a REST server, against a database whose schema may be out of date.
#
# It ships the migrations binary alongside the server and applies pending migrations on start,
# which makes it larger than the base REST image and suited to local development.
FROM docker.io/library/golang:1.26.5-alpine AS builder

ENV CGO_ENABLED=0

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY ./cmd/rest ./cmd/rest
COPY ./cmd/migrations ./cmd/migrations
COPY ./internal/handlers ./internal/handlers
COPY ./internal/dao ./internal/dao
COPY ./internal/core ./internal/core
COPY ./internal/models ./internal/models
COPY ./internal/config ./internal/config

# One RUN so the two binaries share a single warm module + build cache.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags="-s -w" -trimpath -o /rest ./cmd/rest/ && \
    go build -ldflags="-s -w" -trimpath -o /migrations ./cmd/migrations/

FROM docker.io/library/alpine:3.24.1

WORKDIR /

COPY --from=builder /rest /rest
COPY --from=builder /migrations /migrations

# Alpine ships BusyBox wget — no extra package needed for the healthcheck.
HEALTHCHECK --interval=1s --timeout=5s --retries=10 --start-period=1s \
  CMD wget -qO /dev/null http://localhost:8080/ping || exit 1

ENV REST_PORT=8080

# REST port.
EXPOSE 8080

# Apply pending migrations, then start the server.
CMD ["sh", "-c", "/migrations && /rest"]
