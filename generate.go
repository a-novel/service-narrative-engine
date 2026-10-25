package servicetemplate

// Each tool is pinned in its own modfile, so that building it never pulls its
// dependency graph into this module's.

// Generate proto stubs first: mockery type-checks the packages that import them.
//go:generate rm -rf internal/handlers/protogen
//go:generate go tool -modfile=buf.mod buf generate

// Generate mocks.
//go:generate go tool -modfile=mockery.mod mockery
