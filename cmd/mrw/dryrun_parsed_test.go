package main

import (
	"strings"
	"testing"
)

// ADR-060 T1: --dry-run prints each parsed hunk's body count.
func TestDryRunPrintsParsedHunkBodies(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := grepTree(t, map[string]string{"a.go": "package a\nfunc A() {}\n"})
	if _, err := readIn(t, root, "a.go"); err != nil {
		t.Fatal(err)
	}
	doc := "@@ a.go 2 replace anchor=\"func A\" body=1\nfunc A() { _ = 1 }\n"
	out, code := writeIn(t, root, "--dry-run", "--no-check", planFile(t, doc))
	if code != 0 {
		t.Fatalf("dry-run exited %d:\n%s", code, out)
	}
	if !strings.Contains(out, "parsed:") || !strings.Contains(out, "body=1") {
		t.Errorf("dry-run does not print parsed body=1:\n%s", out)
	}
}
