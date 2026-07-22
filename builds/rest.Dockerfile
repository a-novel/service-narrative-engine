# Exposes the service as a REST server.
#
# It needs a database instance with the schema migrations already applied.
FROM docker.io/library/golang:1.26.5-alpine AS builder

ENV CGO_ENABLED=0

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY ./cmd/rest ./cmd/rest
COPY ./internal/handlers ./internal/handlers
COPY ./internal/dao ./internal/dao
COPY ./internal/core ./internal/core
COPY ./internal/models ./internal/models
COPY ./internal/config ./internal/config

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags="-s -w" -trimpath -o /rest ./cmd/rest/

FROM docker.io/library/alpine:3.24.1

WORKDIR /

COPY --from=builder /rest /rest

# Alpine ships BusyBox wget — no extra package needed for the healthcheck.
HEALTHCHECK --interval=1s --timeout=5s --retries=10 --start-period=1s \
  CMD wget -qO /dev/null http://localhost:8080/ping || exit 1

ENV REST_PORT=8080

# REST port.
EXPOSE 8080

CMD ["/rest"]
