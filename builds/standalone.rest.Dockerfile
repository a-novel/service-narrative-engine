# This image exposes our app as a REST server.
#
# It requires a database instance to run properly. The instance may not be patched.
#
# This image will make sure all patches are applied before starting the server. It is a larger
# version of the base REST image, suited for local development rather than full scale production.
FROM docker.io/library/golang:1.26.3-alpine AS builder

WORKDIR /app

# ======================================================================================================================
# Copy build files.
# ======================================================================================================================
COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
COPY "./cmd/rest" "./cmd/rest"
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
RUN go build -o /rest cmd/rest/main.go
RUN go build -o /migrations cmd/migrations/main.go

FROM docker.io/library/alpine:3.23.4

WORKDIR /

COPY --from=builder /rest /rest
COPY --from=builder /migrations /migrations

# ======================================================================================================================
# Healthcheck.
# ======================================================================================================================
RUN apk --update add curl

HEALTHCHECK --interval=1s --timeout=5s --retries=10 --start-period=1s \
  CMD curl -f http://localhost:8080/ping || exit 1

# ======================================================================================================================
# Finish setup.
# ======================================================================================================================
# Make sure the executable uses the default port.
ENV REST_PORT=8080

# REST port.
EXPOSE 8080

# Run patches before starting the server.
CMD ["sh", "-c", "/migrations && /rest"]
