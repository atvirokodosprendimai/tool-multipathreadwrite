package main

import (
	"bytes"
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// TestVersionFlagPrintsTheStampedVersion runs the real root command with
// --version and asserts the output contains the `version` variable.
//
// It exists because of how the release build stamps a version:
//
//	go build -ldflags "-X main.version=v1.2.3" ./cmd/mrw
//
// The linker DISCARDS a -X for a symbol that does not exist, and says nothing.
// So deleting the variable, renaming it, or unwiring it from the command leaves
// a workflow that still builds, still publishes, and ships binaries that report
// "dev" forever. Nothing else would go red. This does.
func TestVersionFlagPrintsTheStampedVersion(t *testing.T) {
	cmd := rootCommand()
	var buf bytes.Buffer
	cmd.Writer = &buf

	if err := cmd.Run(context.Background(), []string{"mrw", "--version"}); err != nil {
		t.Fatalf("running --version: %v", err)
	}
	if got := buf.String(); !strings.Contains(got, version) {
		t.Errorf("--version printed %q, want it to contain the version variable %q", got, version)
	}
}

// TestVersionIsAVariableTheLinkerCanStamp guards the other half, which the test
// above cannot see: -X only works on a package-level STRING VARIABLE. Turning
// `version` into a const, or into any non-string type, makes the stamp a silent
// no-op while every test that merely reads the value keeps passing.
func TestVersionIsAVariableTheLinkerCanStamp(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", nil, 0)
	if err != nil {
		t.Fatalf("parse main.go: %v", err)
	}

	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		decl, ok := n.(*ast.GenDecl)
		if !ok {
			return true
		}
		for _, spec := range decl.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range vs.Names {
				if name.Name != "version" {
					continue
				}
				found = true
				if decl.Tok != token.VAR {
					t.Errorf("%s: `version` is declared with %s; -ldflags -X can only stamp a var, "+
						"so a release build would silently keep the default",
						fset.Position(name.Pos()), decl.Tok)
				}
				if len(vs.Values) == 1 {
					if lit, ok := vs.Values[0].(*ast.BasicLit); !ok || lit.Kind != token.STRING {
						t.Errorf("%s: `version` is not initialised with a string literal; -X stamps "+
							"strings only", fset.Position(name.Pos()))
					}
				}
			}
		}
		return true
	})
	if !found {
		t.Error("main.go declares no `version` identifier — the release workflow stamps " +
			"-X main.version, and the linker discards that silently")
	}
}
