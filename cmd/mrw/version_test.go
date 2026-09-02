package main

import (
	"bytes"
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"runtime/debug"
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

// A LOCAL build must say which commit it came from, because two local builds
// are otherwise indistinguishable and a stale one is dangerous rather than
// merely old: the ./bin/mrw in this checkout was a day behind on 2026-09-02 and
// still had the read-before-modify bypass live, while `mrw --version` reported
// "dev" exactly as the fresh one did.
//
// Driven through versionFrom rather than the real build info, because a TEST
// BINARY carries no vcs.revision — `go test` does not stamp one. A test that
// read the real settings could only ever SKIP, and I wrote that version first:
// it passed, reported SKIP, and asserted nothing.
func TestVersionReportsTheRevisionItWasBuiltFrom(t *testing.T) {
	rev := "9471accd8503d1fd0c29120e954e72483ca682fc"
	for _, tc := range []struct {
		name     string
		settings []debug.BuildSetting
		want     string
	}{
		{
			name:     "clean checkout names the short revision",
			settings: []debug.BuildSetting{{Key: "vcs.revision", Value: rev}, {Key: "vcs.modified", Value: "false"}},
			want:     "v1.2.3 (9471acc)",
		},
		{
			name:     "a dirty tree says so, because the commit alone would be a lie",
			settings: []debug.BuildSetting{{Key: "vcs.revision", Value: rev}, {Key: "vcs.modified", Value: "true"}},
			want:     "v1.2.3 (9471acc, dirty)",
		},
		{
			// A tarball build, or -buildvcs=false. Unstamped is not the same as
			// lying, so it reports the bare version rather than inventing one.
			name:     "no VCS info falls back to the bare version",
			settings: []debug.BuildSetting{{Key: "-buildmode", Value: "exe"}},
			want:     "v1.2.3",
		},
		{
			name:     "a revision shorter than seven characters is not truncated",
			settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "abc"}},
			want:     "v1.2.3 (abc)",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := versionFrom("v1.2.3", tc.settings); got != tc.want {
				t.Errorf("versionFrom = %q, want %q", got, tc.want)
			}
		})
	}
}

// And the wiring: --version must print what versionString composes, or the
// composition above is tested and unreachable.
func TestVersionFlagPrintsWhatVersionStringComposes(t *testing.T) {
	cmd := rootCommand()
	var buf bytes.Buffer
	cmd.Writer = &buf
	if err := cmd.Run(context.Background(), []string{"mrw", "--version"}); err != nil {
		t.Fatalf("running --version: %v", err)
	}
	if got, want := buf.String(), versionString(); !strings.Contains(got, want) {
		t.Errorf("--version printed %q, want it to contain %q", got, want)
	}
}
