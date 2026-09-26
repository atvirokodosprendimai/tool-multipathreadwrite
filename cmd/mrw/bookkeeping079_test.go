package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/authoring"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-079. `mrw seen` loaded the ledger without its lock, and save empties the
// file to rewrite it, so a `mrw seen` beside a writer could print a ledger with
// nothing in it. It reads under the lock, as a writer's snapshot does.
func TestSeenNeverPrintsAHalfSavedLedger(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	obs := map[string]seen.Observation{"a.txt": {SHA: "1"}}
	if err := seen.Record(root, obs); err != nil {
		t.Fatal(err)
	}
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-stop:
				return
			default:
				_ = seen.Record(root, obs)
			}
		}
	}()
	defer func() { close(stop); <-done }()
	for i := 0; i < 400; i++ {
		out, code := runIn(t, root, "seen")
		if code != 0 || !strings.Contains(out, "a.txt") {
			t.Fatalf("run %d: `mrw seen` printed a ledger without a.txt (exit %d):\n%s", i, code, out)
		}
	}
}

// ADR-079. A clean --dry-run was tallied as applied, and counted among the
// landed writes though nothing landed; a refused one is still a refusal.
func TestACleanDryRunRecordsNothing(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readIn(t, root, "a.txt"); err != nil {
		t.Fatal(err)
	}
	if out, code := writeIn(t, root, "--no-check", "--dry-run", planFile(t, "@@ a.txt 1 replace\nb\n")); code != 0 {
		t.Fatalf("clean dry run: exit %d\n%s", code, out)
	}
	if tally, _ := authoring.Load(root); tally.Plans() != 0 {
		t.Errorf("a clean dry run was tallied: %v", tally)
	}
	if _, code := writeIn(t, root, "--no-check", "--dry-run", planFile(t, "@@ a.txt 9 replace\nb\n")); code == 0 {
		t.Fatal("a dry run addressing line 9 of a one-line file applied")
	}
	if tally, _ := authoring.Load(root); tally["refused_apply"] != 1 || tally.Plans() != 1 {
		t.Errorf("a refused dry run is not one refusal: %v", tally)
	}
}
