package core_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// projectWriteLockExempt maps each service that writes a project's content without the project lock to the
// reason it may.
var projectWriteLockExempt = map[string]string{
	"ProjectCreate":     "it creates the project, stamped at creation, with its required layers",
	"ProjectDelete":     "it removes the project",
	"ProjectLogoUpdate": "a logo is cosmetic and leaves the project's edit date",
	"ProjectLogoDelete": "a logo is cosmetic and leaves the project's edit date",
}

// projectWriteOperations end the names of the DAO requests that write a project's content, such as
// dao.BeatInsertRequest or dao.CellsUpsertRequest.
var projectWriteOperations = []string{"Insert", "Update", "Upsert", "Delete", "Move"}

// projectWrites locates the first content write and the first project lock in the Exec method of a service.
// Each position is token.NoPos when the method has none.
type projectWrites struct {
	write token.Pos
	lock  token.Pos
}

// TestProjectWriteLock enforces the project lock rule over internal/core, where it lives. It parses the
// package's non-test files and checks that the Exec method of every service that builds a DAO request writing a
// project's content calls lockOwnedProject before that request. The lock runs the project's writes one at a
// time and stamps the project's edited_at, which orders the project list.
//
// A service listed in projectWriteLockExempt skips the rule, and the test fails once that service no longer
// exists or no longer writes, so the list holds only real exemptions.
func TestProjectWriteLock(t *testing.T) {
	t.Parallel()

	fileSet := token.NewFileSet()
	services := parseProjectWrites(t, fileSet)

	for name, reason := range projectWriteLockExempt {
		t.Run("Exempt/"+name, func(t *testing.T) {
			t.Parallel()

			require.True(t, services[name].write.IsValid(),
				"%s is exempt from the project lock because %s, but no Exec method of internal/core writes "+
					"project content for it: remove the exemption", name, reason)
		})
	}

	for name, service := range services {
		_, exempt := projectWriteLockExempt[name]
		if exempt || !service.write.IsValid() {
			continue
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.True(t, service.lock.IsValid() && service.lock < service.write,
				"%s writes project content at %s without calling lockOwnedProject first. Every write to a "+
					"project's content takes the project lock before its first write, in the same transaction, so "+
					"the project's writes run one at a time and stamp its edited_at. Call lockOwnedProject first, "+
					"or add %s to projectWriteLockExempt with its reason",
				name, fileSet.Position(service.write), name)
		})
	}
}

// parseProjectWrites maps each service of the package to the writes and the lock of its Exec method.
func parseProjectWrites(t *testing.T, fileSet *token.FileSet) map[string]projectWrites {
	t.Helper()

	paths, err := filepath.Glob("*.go")
	require.NoError(t, err)

	services := map[string]projectWrites{}

	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}

		file, err := parser.ParseFile(fileSet, path, nil, parser.SkipObjectResolution)
		require.NoError(t, err)

		for _, decl := range file.Decls {
			name, body, ok := serviceExec(decl)
			if ok {
				services[name] = findProjectWrites(body)
			}
		}
	}

	return services
}

// serviceExec returns the receiver type and the body of an Exec method declared on a pointer receiver.
func serviceExec(decl ast.Decl) (string, *ast.BlockStmt, bool) {
	function, ok := decl.(*ast.FuncDecl)
	if !ok || function.Recv == nil || function.Name.Name != "Exec" || function.Body == nil {
		return "", nil, false
	}

	pointer, ok := function.Recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return "", nil, false
	}

	receiver, ok := pointer.X.(*ast.Ident)
	if !ok {
		return "", nil, false
	}

	return receiver.Name, function.Body, true
}

// findProjectWrites locates the first content write and the first lockOwnedProject call of a method body. The
// walk visits nodes in source order, so the first match of each is the earliest.
func findProjectWrites(body *ast.BlockStmt) projectWrites {
	var found projectWrites

	ast.Inspect(body, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.CallExpr:
			callee, ok := node.Fun.(*ast.Ident)
			if ok && callee.Name == "lockOwnedProject" && !found.lock.IsValid() {
				found.lock = node.Pos()
			}
		case *ast.CompositeLit:
			if isProjectWrite(node) && !found.write.IsValid() {
				found.write = node.Pos()
			}
		}

		return true
	})

	return found
}

// isProjectWrite reports whether a literal builds a DAO request whose operation writes project content.
func isProjectWrite(literal *ast.CompositeLit) bool {
	selector, ok := literal.Type.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	pkg, ok := selector.X.(*ast.Ident)
	if !ok || pkg.Name != "dao" {
		return false
	}

	operation, ok := strings.CutSuffix(selector.Sel.Name, "Request")

	return ok && slices.ContainsFunc(projectWriteOperations, func(suffix string) bool {
		return strings.HasSuffix(operation, suffix)
	})
}
