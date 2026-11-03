# Runs the migrations, then the REST server, in one image for local development and tests.
# A production deployment runs the rest.Dockerfile image after the migrations job instead.
FROM docker.io/library/golang:1.27.3-alpine AS builder

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
COPY ./internal/models/schemas ./internal/models/schemas
COPY ./internal/models/migrations ./internal/models/migrations
COPY ./internal/config ./internal/config

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags="-s -w" -trimpath -o /rest ./cmd/rest/ && \
    go build -ldflags="-s -w" -trimpath -o /migrations ./cmd/migrations/

FROM docker.io/library/alpine:3.24.3

COPY --from=builder /rest /rest
COPY --from=builder /migrations /migrations

# The healthcheck uses the wget of Alpine's BusyBox.
HEALTHCHECK --interval=1s --timeout=5s --retries=10 --start-period=1s \
  CMD ["wget", "-qO", "/dev/null", "http://localhost:8080/v1/ping"]

ENV REST_PORT=8080
# Local, test and consumer stacks call the server directly, with no service token. Production runs the
# rest image, which keeps the issuer required. The variable holds a boolean, not a secret, though its
# name says token.
# hadolint ignore=DL3064
ENV SERVICE_TOKEN_ISSUER_OPTIONAL=true

EXPOSE 8080

CMD ["sh", "-c", "/migrations && /rest"]
