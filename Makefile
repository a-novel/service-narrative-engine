# ================================================================================
# Run tests.
# ================================================================================
test-unit:
	bash -c "set -m; bash '$(CURDIR)/scripts/test.sh'"

test-pkg:
	bash -c "set -m; bash '$(CURDIR)/scripts/test.pkg.sh'"

test-pkg-js:
	bash -c "set -m; bash '$(CURDIR)/scripts/test.pkg.js.sh'"

test: test-unit test-pkg test-pkg-js

# ================================================================================
# Lint.
# ================================================================================
lint-go:
	go tool -modfile=golangci-lint.mod golangci-lint run ./...

lint-proto:
	go tool buf lint

lint-node:
	pnpm lint

lint: lint-go lint-proto lint-node

# ================================================================================
# Format.
# ================================================================================
format-go:
	go mod tidy
	go tool -modfile=golangci-lint.mod golangci-lint run ./... --fix

format-proto:
	go tool buf format -w
	go tool buf dep update

format-node:
	pnpm format

format: format-go format-proto format-node

# ================================================================================
# Generate.
# ================================================================================
generate-go:
	go generate ./...

generate: generate-go

# ================================================================================
# Local dev.
# ================================================================================
run:
	bash -c "set -m; bash '$(CURDIR)/scripts/run.sh'"

build:
	bash -c "set -m; bash '$(CURDIR)/scripts/build.sh'"

install:
	go mod tidy
	pnpm i --frozen-lockfile
