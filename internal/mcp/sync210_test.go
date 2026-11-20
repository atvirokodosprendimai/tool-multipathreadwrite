package mcp

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
)

// ADR-210 T1. A directory sync that fails after the renames is a write that
// landed: mrw_write answers isError with the landed receipt, and under a ceiling
// too small for the receipt the terminal sentence says the same.
func TestMrwWriteReportsASyncThatFailedAfterLanding(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	rs, rf := apply.SyncsDirs, apply.SyncDirFn
	t.Cleanup(func() { apply.SyncsDirs, apply.SyncDirFn = rs, rf })
	apply.SyncsDirs = true
	apply.SyncDirFn = func(*os.File) error { return errors.New("injected sync failure") }
	const words = "not known to be on stable storage"

	res := call(t, root, "mrw_write", map[string]any{"plan": "@@ new.txt 0 create\nhello\n"})
	if res["isError"] != true {
		t.Fatalf("a write whose sync failed after landing was not an error: %v", res)
	}
	var sc struct {
		Applied bool   `json:"applied"`
		Error   string `json:"error"`
	}
	if raw, ok := res["structuredContent"].(map[string]any); ok {
		sc.Applied, _ = raw["applied"].(bool)
		sc.Error, _ = raw["error"].(string)
	}
	if !sc.Applied || !strings.Contains(sc.Error, words) {
		t.Errorf("structured applied=%v error=%q; want the landed receipt with the sync error", sc.Applied, sc.Error)
	}
	if b, err := os.ReadFile(filepath.Join(root, "new.txt")); err != nil || string(b) != "hello\n" {
		t.Errorf("new.txt = %q (%v)", b, err)
	}

	// The terminal sentence: a ceiling the receipt does not fit but the sentence
	// and the sync's words do (they are advisory, outside the write floor).
	restore := MaxResultChars
	t.Cleanup(func() { MaxResultChars = restore })
	saw := false
	for c := minWriteCeiling(); c < minWriteCeiling()+400; c += 5 {
		if err := os.Remove(filepath.Join(root, "new.txt")); err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		MaxResultChars = c
		res = call(t, root, "mrw_write", map[string]any{"plan": "@@ new.txt 0 create\nhello\n"})
		if _, structured := res["structuredContent"]; structured {
			break
		}
		if text := firstText(t, res); strings.Contains(text, "the plan APPLIED") && strings.Contains(text, words) {
			saw = true
		}
	}
	if !saw {
		t.Errorf("no ceiling between the floor and +400 gave a terminal sentence that says %q", words)
	}
}

// ADR-210 (Zy, 2026-11-20). A volume that cannot flush a directory or drain is not
// a failure: mrw_write answers a clean landing, not an error, and the structured
// receipt carries sync_note.
func TestMrwWriteReportsAVolumeWithNoBarrierAsANote(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	rs, rf := apply.SyncsDirs, apply.SyncDirFn
	t.Cleanup(func() { apply.SyncsDirs, apply.SyncDirFn = rs, rf })
	apply.SyncsDirs = true
	apply.SyncDirFn = func(*os.File) error { return apply.ErrNoBarrier }
	res := call(t, root, "mrw_write", map[string]any{"plan": "@@ new.txt 0 create\nhello\n"})
	sc, _ := res["structuredContent"].(map[string]any)
	note, _ := sc["sync_note"].(string)
	if res["isError"] == true || sc["applied"] != true || !strings.Contains(note, "no device barrier") {
		t.Errorf("isError=%v applied=%v sync_note=%q; want a clean landing carrying the note", res["isError"], sc["applied"], note)
	}
}
