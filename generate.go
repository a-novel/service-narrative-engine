package authentication

// Generate proto stubs first (mockery depends on compiled packages).
//go:generate rm -rf internal/handlers/protogen
//go:generate go tool buf generate

// Generate mocks.
//go:generate go tool mockery
