package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// partialTree returns a checkout whose plan partialPlan commits partially:
// a.txt's replace lands, then the unlink in the read-only d/ fails.
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
	writeFiles(t, root, map[string]string{"a.txt": "a\n", "d/x.txt": "x\n"})
	if out, code := runIn(t, root, "read", "a.txt", "d/x.txt"); code != 0 {
		t.Fatalf("read: exit %d:\n%s", code, out)
	}
	d := filepath.Join(root, "d")
	if err := os.Chmod(d, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(d, 0o755) })
	return root
}

const partialPlan = "@@ a.txt 1 replace\nA\n@@ d/x.txt - unlink\n"

// ADR-102 T1. A commit that failed after a file landed was counted
// refused_apply, so `landed writes` missed a tree that had changed. It is one
// partially_applied, counted in landed, and stats prints the name.
func TestAPartialCommitIsCountedAsPartiallyApplied(t *testing.T) {
	root := partialTree(t)
	out, code := writeIn(t, root, "--no-check", planFile(t, partialPlan))
	if code != exitUsage || !strings.Contains(out, "PARTIALLY APPLIED") {
		t.Fatalf("exit %d, want %d and a PARTIALLY APPLIED receipt:\n%s", code, exitUsage, out)
	}
	if b, _ := os.ReadFile(filepath.Join(root, "a.txt")); string(b) != "A\n" {
		t.Fatalf("a.txt did not land: %q", b)
	}
	js, code := runIn(t, root, "stats", "--json")
	var s struct {
		Counts map[string]int `json:"counts"`
		Landed int            `json:"landed"`
	}
	if err := json.Unmarshal([]byte(js), &s); code != 0 || err != nil {
		t.Fatalf("stats --json: exit %d, %v:\n%s", code, err, js)
	}
	if s.Counts["partially_applied"] != 1 || s.Counts["refused_apply"] != 0 || s.Landed != 1 {
		t.Errorf("counts %v landed %d, want partially_applied 1, refused_apply 0, landed 1", s.Counts, s.Landed)
	}
	if txt, _ := runIn(t, root, "stats"); !strings.Contains(txt, "partially_applied") {
		t.Errorf("stats does not print partially_applied:\n%s", txt)
	}
}
