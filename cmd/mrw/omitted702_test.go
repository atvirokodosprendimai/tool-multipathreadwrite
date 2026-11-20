package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// #702. A checkout below a directory whose name holds a line feed has no ledger
// key that can be saved, so a Record there leaves every record out and returns
// nil, as it always did. Only a --quiet read, which serves no line, asks the
// ledger afterwards and refuses what it cannot hold; a plain read, a write and
// a --force --sub plan go on as on main.
func lineFeedRoot(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("a line feed is not a legal Windows file name")
	}
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	// A ledger key is the root's path below the checkout's top plus the file's
	// (seen.storedKey), so the line feed must sit below a .git.
	top := t.TempDir()
	if err := os.MkdirAll(filepath.Join(top, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(top, "sub\ndir")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestAQuietReadTheLedgerCannotHoldExitsTwoAndAPlainReadExitsZero(t *testing.T) {
	root := lineFeedRoot(t)
	var out string
	var code int
	errOut := stderrOf(t, func() { out, code = runIn(t, root, "read", "--quiet", "f.txt") })
	if code != 2 || !strings.Contains(errOut+out, "line feed") {
		t.Errorf("a --quiet read exited %d with %q / %q, want 2 naming the line feed", code, out, errOut)
	}
	// Zero lines to license: nothing to keep, so the read ends as it would anywhere.
	if _, code = runIn(t, root, "read", "--quiet", "f.txt:99"); code != 1 {
		t.Errorf("a --quiet read of a range past the end exited %d, want 1", code)
	}
	out, code = runIn(t, root, "read", "f.txt")
	if code != 0 || !strings.Contains(out, "hello") {
		t.Errorf("a plain read exited %d with %q, want 0 and its text", code, out)
	}
}

func TestALandedWriteUnderAnUnrecordableRootKeepsItsExitAndRunsItsCheck(t *testing.T) {
	root := lineFeedRoot(t)
	if err := os.WriteFile(filepath.Join(root, ".quality-harness.json"), []byte(`{"check":"echo ran > marker"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := filepath.Join(t.TempDir(), "p.plan")
	if err := os.WriteFile(plan, []byte("@@ new.txt 0 create\nhello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := runIn(t, root, "write", "--check", plan)
	if code != 0 {
		t.Errorf("a landed write exited %d, want 0: %s", code, out)
	}
	if _, err := os.Stat(filepath.Join(root, "marker")); err != nil {
		t.Errorf("the check never ran: %v", err)
	}
}

func TestAForcedSubPlanUnderAnUnrecordableRootApplies(t *testing.T) {
	root := lineFeedRoot(t)
	out, code := runIn(t, root, "write", "--force", "--grep", "hello", "--sub", "bye", "f.txt")
	if got, _ := os.ReadFile(filepath.Join(root, "f.txt")); code != 0 || string(got) != "bye\n" {
		t.Errorf("--force --grep --sub exited %d, file %q: %s", code, got, out)
	}
}
