# This image exposes our app as a gRPC server.
#
# It requires a database instance to run properly. The instance may not be patched.
#
# This image will make sure all patches are applied before starting the server. It is a larger
# version of the base gRPC image, suited for local development rather than full scale production.
FROM docker.io/library/golang:1.26.4-alpine AS builder

WORKDIR /app

# ======================================================================================================================
# Copy build files.
# ======================================================================================================================
COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
COPY "./cmd/grpc" "./cmd/grpc"
COPY "./cmd/migrations" "./cmd/migrations"
COPY ./internal/handlers ./internal/handlers
COPY ./internal/dao ./internal/dao
COPY ./internal/services ./internal/services
COPY ./internal/models ./internal/models
COPY ./internal/config ./internal/config

RUN go mod download

# ======================================================================================================================
# Build executables.
# ======================================================================================================================
RUN go build -o /grpc cmd/grpc/main.go
RUN go build -o /migrations cmd/migrations/main.go

# Used for healthcheck.
RUN GOBIN=/grpcurl go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

FROM docker.io/library/alpine:3.23.4

WORKDIR /

COPY --from=builder /grpc /grpc
COPY --from=builder /migrations /migrations

COPY --from=builder /grpcurl /bin/

# ======================================================================================================================
# Healthcheck.
# ======================================================================================================================
HEALTHCHECK --interval=1s --timeout=5s --retries=10 --start-period=1s \
  CMD grpcurl --plaintext -d '' localhost:8080 grpc.health.v1.Health/Check || exit 1

# ======================================================================================================================
# Finish setup.
# ======================================================================================================================
# Make sure the executable uses the default port.
ENV GRPC_PORT=8080

# GRPC port.
EXPOSE 8080
# TLS port.
EXPOSE 443

# Run patches before starting the server.
CMD ["sh", "-c", "/migrations && /grpc"]
