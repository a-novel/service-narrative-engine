# ================================================================================
# Run tests.
# ================================================================================
test-unit:
	bash -c "set -m; bash '$(CURDIR)/scripts/test.sh'"

# Short tests skip the tests that call AI, for less consumption.
test-unit-short:
	bash -c "set -m; bash '$(CURDIR)/scripts/test.sh' -short"

test-pkg-js:
	bash -c "set -m; bash '$(CURDIR)/scripts/test.pkg.js.sh'"

test: test-unit # test-pkg-js

test-short: test-unit-short

# ================================================================================
# Lint.
# ================================================================================
lint-go:
	go tool golangci-lint run

lint-node:
	pnpm lint

lint: lint-go lint-node

# ================================================================================
# Format.
# ================================================================================
format-go:
	go mod tidy
	go tool golangci-lint run --fix

format-node:
	pnpm format

format: format-go format-node

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
