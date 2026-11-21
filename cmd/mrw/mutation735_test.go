package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// mutationOf runs `mrw write --json` and returns the receipt's mutation and the
// exit code. The key is read as a string, so an absent one reads "".
func mutationOf(t *testing.T, root string, args ...string) (mutation string, code int, out string) {
	t.Helper()
	out, code = writeIn(t, root, append([]string{"--no-check", "--json"}, args...)...)
	var r struct {
		Mutation string `json:"mutation"`
	}
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatalf("exit %d, stdout is not one JSON document: %v\n%s", code, err, out)
	}
	return r.Mutation, code, out
}

// primedTree is a checkout holding a.go, read whole so a plan may edit it.
func primedTree(t *testing.T) string {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	writeFiles(t, root, map[string]string{"a.go": "package a\n"})
	if out, code := runIn(t, root, "read", "a.go"); code != 0 {
		t.Fatalf("read: exit %d:\n%s", code, out)
	}
	return root
}

// Issue #735. Exit 2 could not say whether resending the plan was safe: the
// engine knows how much of it reached the tree (writer.MutationOf), and no
// receipt said. `mutation` is none, partial or complete on every outcome.
func TestTheWriteReceiptSaysHowMuchOfThePlanReachedTheTree(t *testing.T) {
	t.Run("a landed write is complete, exit 0", func(t *testing.T) {
		root := primedTree(t)
		if m, code, out := mutationOf(t, root, planFile(t, "@@ a.go 1 replace\npackage b\n")); m != "complete" || code != 0 {
			t.Errorf("mutation %q exit %d, want complete and 0:\n%s", m, code, out)
		}
	})
	t.Run("a dry run is none", func(t *testing.T) {
		root := primedTree(t)
		if m, code, out := mutationOf(t, root, "--dry-run", planFile(t, "@@ a.go 1 replace\npackage b\n")); m != "none" || code != 0 {
			t.Errorf("mutation %q exit %d, want none and 0:\n%s", m, code, out)
		}
	})
	t.Run("a refused plan is none, exit 1", func(t *testing.T) {
		root := primedTree(t)
		if m, code, out := mutationOf(t, root, planFile(t, "@@ a.go 9 replace\npackage b\n")); m != "none" || code != exitNotApplied {
			t.Errorf("mutation %q exit %d, want none and %d:\n%s", m, code, exitNotApplied, out)
		}
	})
	t.Run("a plan refused after its text was read is none, and still says so", func(t *testing.T) {
		root := primedTree(t)
		if m, code, out := mutationOf(t, root, planFile(t, "this is not a plan\n")); m != "none" || code == 0 {
			t.Errorf("mutation %q exit %d, want none and a refusal:\n%s", m, code, out)
		}
	})
	t.Run("a commit that failed after a file landed and put it back is none, exit 2", func(t *testing.T) {
		root := partialTree(t)
		if m, code, out := mutationOf(t, root, planFile(t, partialPlan)); m != "none" || code != exitUsage {
			t.Errorf("mutation %q exit %d, want none and %d:\n%s", m, code, exitUsage, out)
		}
	})
	t.Run("a ledger that could not record a landing is complete, exit 2", func(t *testing.T) {
		if runtime.GOOS == "windows" || os.Geteuid() == 0 {
			t.Skip("a read-only file does not stop this user from writing it")
		}
		root := primedTree(t)
		ledger, err := seen.ReadPath(root)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(ledger, 0o444); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(ledger, 0o600) })
		m, code, out := mutationOf(t, root, planFile(t, "@@ a.go 1 replace\npackage b\n"))
		if b, _ := os.ReadFile(filepath.Join(root, "a.go")); string(b) != "package b\n" {
			t.Fatalf("the write did not land, so the ledger was never reached:\n%s", out)
		}
		if m != "complete" || code != exitUsage {
			t.Errorf("mutation %q exit %d, want complete and %d (the tree changed; do not resend):\n%s", m, code, exitUsage, out)
		}
	})
}
