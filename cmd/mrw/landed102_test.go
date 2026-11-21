package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// partialTree returns a checkout whose plan partialPlan fails at commit: a.go's
// single-line replace lands, then the unlink in the read-only d/ fails and, since
// ADR-209, a.go is put back.
func partialTree(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("a read-only directory does not stop this user from writing it")
	}
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFiles(t, root, map[string]string{"a.go": "package a\n", "d/x.txt": "x\n"})
	if out, code := runIn(t, root, "read", "a.go", "d/x.txt"); code != 0 {
		t.Fatalf("read: exit %d:\n%s", code, out)
	}
	d := filepath.Join(root, "d")
	if err := os.Chmod(d, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(d, 0o755) })
	return root
}

const partialPlan = "@@ a.go 1 replace\npackage b\n@@ d/x.txt - unlink\n"

// ADR-102 T1, since ADR-209. A commit that failed after a file landed was counted
// refused_apply, then partially_applied; it puts the file back now, so the tree is
// unchanged and stats counts it refused_apply and not landed. (partially_applied is
// left to an undo that could not finish, which the built binary cannot be made to
// reach: TestTheCLIPutsTheContentFilesBackWhenACommitFails drives the whole CLI case.)
func TestAPartialCommitIsCountedAsPartiallyApplied(t *testing.T) {
	root := partialTree(t)
	out, code := writeIn(t, root, "--no-check", planFile(t, partialPlan))
	if code != exitUsage || !strings.Contains(out, "NOTHING WRITTEN") {
		t.Fatalf("exit %d, want %d and a NOTHING WRITTEN receipt:\n%s", code, exitUsage, out)
	}
	js, code := runIn(t, root, "stats", "--json")
	var s struct {
		Counts map[string]int `json:"counts"`
		Landed int            `json:"landed"`
	}
	if err := json.Unmarshal([]byte(js), &s); code != 0 || err != nil {
		t.Fatalf("stats --json: exit %d, %v:\n%s", code, err, js)
	}
	if s.Counts["refused_apply"] != 1 || s.Counts["partially_applied"] != 0 || s.Landed != 0 {
		t.Errorf("counts %v landed %d, want refused_apply 1, partially_applied 0, landed 0", s.Counts, s.Landed)
	}
}
