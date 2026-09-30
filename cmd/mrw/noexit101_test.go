package main

import (
	"testing"
)

// ADR-101. In a tree with no check, `check --json` and the `check` block of
// `write --check --json` said "exit_code": 0 beside "ran": false, and mrw
// exited 2. A check that did not run now says -1 in both receipts.
func TestACheckThatDidNotRunReportsNoExitCodeOfZero(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	writeFiles(t, root, map[string]string{"a.txt": "a\n", "p.mrw": "@@ a.txt 1 replace\nb\n"})
	stdout, err := runSplit(t, "-C", root, "check", "--json", "--full")
	doc, ok := oneDocument(stdout)
	if err == nil || exitCode(err) != exitUsage || !ok || doc["ran"] != false || doc["exit_code"] != float64(-1) {
		t.Errorf("check --json with no check: err %v, want exit %d and one document with ran false, exit_code -1:\n%s", err, exitUsage, stdout)
	}
	if out, code := runIn(t, root, "read", "a.txt"); code != 0 {
		t.Fatalf("read: exit %d:\n%s", code, out)
	}
	t.Chdir(root) // a plan file resolves from the working directory
	stdout, err = runSplit(t, "-C", root, "write", "--check", "--json", "p.mrw")
	doc, ok = oneDocument(stdout)
	c, _ := doc["check"].(map[string]any)
	if err == nil || exitCode(err) != exitUsage || !ok || c["ran"] != false || c["exit_code"] != float64(-1) {
		t.Errorf("write --check --json with no check: err %v, want exit %d and a check block with ran false, exit_code -1:\n%s", err, exitUsage, stdout)
	}
}
