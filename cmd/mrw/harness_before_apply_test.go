package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// badHarness is checkTree with a .quality-harness.json nobody can parse.
func badHarness(t *testing.T) string {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := checkTree(t)
	// Both files are read, so the ledger licenses the edits and the only
	// thing left to refuse them is the harness.
	if _, err := readIn(t, root, "a.go", "notes.md"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".quality-harness.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func fileIs(t *testing.T, root, name, want string) {
	t.Helper()
	got, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("%s is %q, want %q — a refused write changed it", name, got, want)
	}
}

// ADR-072 T1, the Enforced-by. A malformed .quality-harness.json was read
// AFTER the write committed: the tree changed and mrw exited 2 with only the
// JSON error, the status documented as "usage or filesystem". It is read
// first, and it refuses before anything is written.
func TestAMalformedHarnessRefusesTheWriteBeforeAnythingIsWritten(t *testing.T) {
	root := badHarness(t)
	out, code := runIn(t, root, "write", planFile(t, goPlan))
	if code != exitUsage || !strings.Contains(out, ".quality-harness.json") || !strings.Contains(out, "nothing was written") {
		t.Errorf("exit %d, want %d naming the harness and saying nothing was written:\n%s", code, exitUsage, out)
	}
	fileIs(t, root, "a.go", "package a\nfunc A() {}\n")
}

// A prose plan was broken by a bad harness too: the load ran before the prose
// gate. It is refused the same way, before the write.
func TestAMarkdownOnlyPlanIsAlsoRefusedByABadHarness(t *testing.T) {
	root := badHarness(t)
	if out, code := runIn(t, root, "write", planFile(t, mdPlan)); code != exitUsage {
		t.Errorf("exit %d, want %d:\n%s", code, exitUsage, out)
	}
	fileIs(t, root, "notes.md", "# notes\nline two\n")
}

// The pairs: --no-check never reads the harness, and applies.
func TestNoCheckSkipsTheHarnessEntirely(t *testing.T) {
	root := badHarness(t)
	if out, code := runIn(t, root, "write", "--no-check", planFile(t, goPlan)); code != 0 {
		t.Fatalf("--no-check exited %d with a bad harness:\n%s", code, out)
	}
	fileIs(t, root, "a.go", "package a\nfunc A() { _ = 1 }\n")
}

// --dry-run writes nothing and checks nothing, so it never reads the harness.
func TestDryRunDoesNotLoadTheHarness(t *testing.T) {
	root := badHarness(t)
	if out, code := runIn(t, root, "write", "--dry-run", planFile(t, goPlan)); code != 0 {
		t.Fatalf("--dry-run exited %d with a bad harness:\n%s", code, out)
	}
	fileIs(t, root, "a.go", "package a\nfunc A() {}\n")
}
