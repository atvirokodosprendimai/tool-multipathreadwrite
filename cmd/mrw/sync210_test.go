package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// failDirSync makes every directory sync after landing fail, on every platform
// (Windows syncs none of its own accord), for one test.
func failDirSync(t *testing.T) {
	t.Helper()
	rs, rf := apply.SyncsDirs, apply.SyncDirFn
	t.Cleanup(func() { apply.SyncsDirs, apply.SyncDirFn = rs, rf })
	apply.SyncsDirs = true
	apply.SyncDirFn = func(*os.File) error { return errors.New("injected sync failure") }
}

// ADR-210 T1. A directory sync that fails after the renames is a write that
// landed: exit 2, applied with an error naming the path and saying the files are
// written but not known to be on stable storage, the ledger still records the
// file, no check runs and the steps are reported not run, the tally counts it as
// a landing whose check did not run, and a ledger that also failed is named too.
func TestTheCLIReportsASyncThatFailedAfterLandingAsAWriteThatLanded(t *testing.T) {
	t.Setenv("MRW_SHELL", "on")
	setup := func(t *testing.T) string {
		t.Helper()
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		root := t.TempDir()
		writeFiles(t, root, map[string]string{"a.go": "package a\nfunc A() {}\n", ".quality-harness.json": `{"check":"exit 0"}`})
		if out, code := runIn(t, root, "read", "a.go"); code != 0 {
			t.Fatalf("read: exit %d:\n%s", code, out)
		}
		failDirSync(t)
		return root
	}
	doc := func(t *testing.T, out string) map[string]any {
		t.Helper()
		var d map[string]any
		if err := json.NewDecoder(strings.NewReader(out)).Decode(&d); err != nil {
			t.Fatalf("no receipt (%v):\n%s", err, out)
		}
		return d
	}
	counts := func(t *testing.T, root string) map[string]int {
		t.Helper()
		out, _ := runIn(t, root, "stats", "--json")
		var s struct {
			Counts map[string]int `json:"counts"`
		}
		if err := json.Unmarshal([]byte(out), &s); err != nil {
			t.Fatalf("stats: %v\n%s", err, out)
		}
		return s.Counts
	}
	const words = "not known to be on stable storage"

	t.Run("no_check", func(t *testing.T) {
		root := setup(t)
		out, code := runIn(t, root, "write", "--json", "--no-check", planFile(t, goPlan))
		d := doc(t, out)
		msg, _ := d["error"].(string)
		if code != 2 || d["applied"] != true || !strings.Contains(msg, words) || !strings.Contains(msg, "could not sync") {
			t.Fatalf("exit %d applied=%v error=%q; want exit 2, applied, the sync error", code, d["applied"], msg)
		}
		if b, _ := os.ReadFile(filepath.Join(root, "a.go")); !strings.Contains(string(b), "_ = 1") {
			t.Errorf("a.go was not written: %q", b)
		}
		if l, _ := seen.Load(root); l["a.go"].SHA == "" || !l["a.go"].Written {
			t.Errorf("the ledger does not record the written file: %v", l)
		}
		if c := counts(t, root); c["applied"] != 1 || c["refused_apply"] != 0 {
			t.Errorf("tally %v, want one applied and no refusal", c)
		}
	})
	t.Run("check_and_steps_not_run", func(t *testing.T) {
		root := setup(t)
		out, code := runIn(t, root, "write", "--json", "--check", "--then-sh", "exit 0", planFile(t, goPlan))
		d := doc(t, out)
		if code != 2 || d["applied"] != true {
			t.Fatalf("exit %d applied=%v; want exit 2, applied", code, d["applied"])
		}
		if c, ok := d["check"].(map[string]any); ok && c["ran"] == true {
			t.Errorf("the check ran after a sync failure: %v", c)
		}
		then, _ := d["then"].(map[string]any)
		steps, _ := then["steps"].([]any)
		if len(steps) != 1 || steps[0].(map[string]any)["status"] != "not_run" {
			t.Errorf("the step is not reported not_run: %v", then)
		}
		if c := counts(t, root); c["check_not_run"] != 1 || c["refused_apply"] != 0 {
			t.Errorf("tally %v, want check_not_run=1", c)
		}
	})
	t.Run("ledger_fails_too", func(t *testing.T) {
		root := setup(t)
		ledger, err := seen.ReadPath(root)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(ledger, 0o444); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(ledger, 0o600) })
		out, code := runIn(t, root, "write", "--json", "--no-check", planFile(t, goPlan))
		msg, _ := doc(t, out)["error"].(string)
		if code != 2 || !strings.Contains(msg, words) || !strings.Contains(msg, "read ledger could not be updated") {
			t.Errorf("exit %d error=%q; want both the sync and the ledger failure named", code, msg)
		}
	})
	// A volume that cannot flush a directory or drain (ErrNoBarrier) is not a
	// failure (Zy, 2026-11-20): the write is a clean landing, exit 0, the JSON
	// receipt carries sync_note, and the human form says it once on stderr.
	t.Run("no_barrier_proceeds_with_a_note", func(t *testing.T) {
		root := setup(t)
		apply.SyncDirFn = func(*os.File) error { return apply.ErrNoBarrier }
		out, code := runIn(t, root, "write", "--json", "--no-check", planFile(t, goPlan))
		d := doc(t, out)
		note, _ := d["sync_note"].(string)
		if code != 0 || d["applied"] != true || d["error"] != nil || !strings.Contains(note, "no device barrier") {
			t.Fatalf("exit %d applied=%v error=%v sync_note=%q; want a clean landing with the note", code, d["applied"], d["error"], note)
		}
		if l, _ := seen.Load(root); !l["a.go"].Written {
			t.Errorf("the ledger does not record the written file: %v", l)
		}
	})
	t.Run("no_barrier_is_said_once_in_the_human_form", func(t *testing.T) {
		root := setup(t)
		apply.SyncDirFn = func(*os.File) error { return apply.ErrNoBarrier }
		var code int
		errOut := stderrOf(t, func() {
			_, code = runIn(t, root, "write", "--no-check", planFile(t, goPlan))
		})
		if code != 0 || strings.Count(errOut, "no device barrier") != 1 {
			t.Errorf("exit %d, the line appears %d time(s) on stderr; want a clean landing saying it once:\n%s", code, strings.Count(errOut, "no device barrier"), errOut)
		}
	})
}
