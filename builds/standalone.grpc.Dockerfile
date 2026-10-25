# Runs the migrations, then the gRPC server, in one image for local development and tests.
# A production deployment runs the grpc.Dockerfile image after the migrations job instead.
FROM docker.io/library/golang:1.27.3-alpine AS builder

ENV CGO_ENABLED=0

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# grpcurl backs the healthcheck; installing it before the source copy keeps its layer cached too.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOBIN=/usr/local/bin go install github.com/fullstorydev/grpcurl/cmd/grpcurl@v1.9.3

COPY ./cmd/grpc ./cmd/grpc
COPY ./cmd/migrations ./cmd/migrations
COPY ./internal/handlers ./internal/handlers
COPY ./internal/dao ./internal/dao
COPY ./internal/core ./internal/core
COPY ./internal/models/migrations ./internal/models/migrations
COPY ./internal/config ./internal/config

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags="-s -w" -trimpath -o /grpc ./cmd/grpc/ && \
    go build -ldflags="-s -w" -trimpath -o /migrations ./cmd/migrations/

FROM docker.io/library/alpine:3.24.3

COPY --from=builder /grpc /grpc
COPY --from=builder /migrations /migrations
COPY --from=builder /usr/local/bin/grpcurl /usr/local/bin/grpcurl

HEALTHCHECK --interval=1s --timeout=5s --retries=10 --start-period=1s \
  CMD ["grpcurl", "--plaintext", "-d", "", "localhost:8080", "grpc.health.v1.Health/Check"]

ENV GRPC_PORT=8080

EXPOSE 8080
# TLS port.
EXPOSE 443

CMD ["sh", "-c", "/migrations && /grpc"]
