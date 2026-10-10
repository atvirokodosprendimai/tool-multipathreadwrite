package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-143. The CLI refuses a write into .git — --create and a plan alike — with
// exit 1 and makes nothing; a read of the same path is served.
func TestWriteIntoDotGitIsRefusedAndReadIsNot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "config"), []byte("[core]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out string
	var code int
	withStdin(t, "#!/bin/sh\n", func() { out, code = runIn(t, root, "write", "--no-check", "--create", ".git/hooks/pre-commit") })
	if code != 1 || !strings.Contains(out, "inside a .git directory") {
		t.Errorf("--create into .git: exit %d\n%s\nwant exit 1 naming .git", code, out)
	}
	if _, err := os.Stat(filepath.Join(root, ".git", "hooks")); err == nil {
		t.Error("a refused --create made .git/hooks")
	}
	out, code = runIn(t, root, "read", ".git/config")
	if code != 0 || !strings.Contains(out, "[core]") {
		t.Errorf("read of .git/config: exit %d\n%s\nwant it served", code, out)
	}
	withStdin(t, "@@ .git/config 1 replace\n[core]\n\thooksPath = x\n", func() { out, code = runIn(t, root, "write", "--no-check", "-") })
	if code != 1 || !strings.Contains(out, "inside a .git directory") {
		t.Errorf("plan edit of .git/config: exit %d\n%s\nwant exit 1 naming .git", code, out)
	}
	if b, _ := os.ReadFile(filepath.Join(root, ".git", "config")); string(b) != "[core]\n" {
		t.Errorf(".git/config = %q after a refused plan", b)
	}
}
