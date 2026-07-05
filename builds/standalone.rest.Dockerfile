# This image exposes our app as a REST server.
#
# It requires a database instance to run properly. The instance may not be patched.
#
# This image will make sure all patches are applied before starting the server. It is a larger
# version of the base REST image, suited for local development rather than full scale production.
FROM docker.io/library/golang:1.26.4-alpine AS builder

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

# Make sure the executable uses the default port.
ENV REST_PORT=8080

# REST port.
EXPOSE 8080

# Run patches before starting the server.
CMD ["sh", "-c", "/migrations && /rest"]
