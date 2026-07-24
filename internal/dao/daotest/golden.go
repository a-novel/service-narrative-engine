package daotest

import (
	"os"
	"path/filepath"
	"testing"
)

// Golden reads a fixture from the calling package's testdata directory. Go runs a test with its own
// package directory as the working directory, so the path resolves the same way from every package.
//
// Fixture bodies are stored pretty-printed, so changing one shows up in review as the lines that
// changed rather than one long line swapped for another.
func Golden(t *testing.T, name string) string {
	t.Helper()

	// The path is a literal directory joined with a name the test wrote itself. There is no
	// untrusted input here to sanitise.
	data, err := os.ReadFile(filepath.Join("testdata", name)) //nolint:gosec // Test-supplied path.
	if err != nil {
		panic(err)
	}

	return string(data)
}
