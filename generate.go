package servicenarrativeengine

// Each tool is pinned in its own modfile, so that building it never pulls its
// dependency graph into this module's.

// Generate mocks.
//go:generate go tool -modfile=mockery.mod mockery
