# This image exposes our app as a gRPC server.
#
# It requires a patched database instance to run properly.
FROM docker.io/library/golang:1.26.3-alpine AS builder

WORKDIR /app

# ======================================================================================================================
# Copy build files.
# ======================================================================================================================
COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
COPY "./cmd/grpc" "./cmd/grpc"
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

# Used for healthcheck.
RUN GOBIN=/grpcurl go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

FROM docker.io/library/alpine:3.23.4

WORKDIR /

COPY --from=builder /grpc /grpc

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

CMD ["/grpc"]
