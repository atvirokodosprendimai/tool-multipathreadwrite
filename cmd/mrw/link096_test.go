package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-096 T1. `mrw read --grep P dlink`, where dlink is an in-root link to a
// directory, answered "no file matched" with no line about dlink. It is now a
// REFUSED line naming the directory to name instead, and it counts.
func TestGrepRefusesANamedDirectoryLinkByName(t *testing.T) {
	root := grepTree(t, map[string]string{"d/f.go": "package d\nWANTED\n"})
	if err := os.Symlink("d", filepath.Join(root, "dlink")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	out, err := readIn(t, root, "--grep", "WANTED", "dlink")
	if err == nil {
		t.Fatalf("a named directory link alone exited 0:\n%s", out)
	}
	if !strings.Contains(out, "==> dlink  REFUSED") || !strings.Contains(out, "name d") {
		t.Errorf("no REFUSED line naming dlink and the directory d:\n%s", out)
	}
	if !strings.Contains(errString(err), "no file matched") {
		t.Errorf("err = %v; want no file matched", err)
	}

	out, err = readIn(t, root, "--grep", "WANTED", "d", "dlink")
	if err == nil {
		t.Fatalf("a directory link named beside d exited 0:\n%s", out)
	}
	if !strings.Contains(out, "==> d/f.go") {
		t.Errorf("d's file was not served beside the refusal:\n%s", out)
	}
	if !strings.Contains(out, "==> dlink  REFUSED") {
		t.Errorf("no REFUSED line for dlink beside d:\n%s", out)
	}
	if !strings.Contains(errString(err), "1 range(s) could not be served") {
		t.Errorf("err = %v; want one range not served", err)
	}
}
