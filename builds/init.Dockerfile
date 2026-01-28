# This image runs a job that will create / update a default super-admin user.
#
# It requires a patched database instance to run properly.
FROM docker.io/library/golang:1.25.6-alpine AS builder

WORKDIR /app

# ======================================================================================================================
# Copy build files.
# ======================================================================================================================
COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
COPY "./cmd/init" "./cmd/init"
COPY ./internal/config ./internal/config
COPY ./internal/dao ./internal/dao
COPY ./internal/services ./internal/services
COPY ./internal/lib ./internal/lib
COPY ./internal/models ./internal/models

RUN go mod download

# ======================================================================================================================
# Build executables.
# ======================================================================================================================
RUN go build -o /init cmd/init/main.go

FROM docker.io/library/node:24.13.0-alpine AS version

COPY ./package.json ./package.json

RUN touch /version
RUN echo $(node -p "require('./package.json').version") >> /version

FROM docker.io/library/alpine:3.23.3

WORKDIR /

COPY --from=builder /init /init
COPY --from=version /version /version

CMD ["sh", "-c", "VERSION=$(cat /version) /init"]
