package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-077. An ast-grep hit inside mrw's own state is dropped as the walk drops
// every discovered path the boundary refuses (ADR-007 rule 2), rather than
// printed REFUSED at exit 1; the caller's hit beside it is served.
func TestAstGrepDropsAHitInsideMrwsState(t *testing.T) {
	root := grepTree(t, map[string]string{
		"hit.go":         "package hit\nfunc Target() {}\n",
		".st/mrw/k/seen": "func Target() {}\n",
	})
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, ".st"))
	installFakeAstGrep(t, `[{"file":".st/mrw/k/seen","range":{"start":{"line":0},"end":{"line":0}}},{"file":"hit.go","range":{"start":{"line":1},"end":{"line":1}}}]`, 0)
	out, err := readIn(t, root, "--ast-grep", "func Target")
	if err != nil {
		t.Fatalf("a hit inside mrw's state failed the read: %v\n%s", err, out)
	}
	if !strings.Contains(out, "==> hit.go") || strings.Contains(out, ".st/mrw") {
		t.Errorf("want hit.go served and nothing of mrw's state:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(root, ".st", "mrw", "k", "seen")); err != nil {
		t.Fatal(err)
	}
}

// Codex review of #238. A --files-from list and a plan file are shell
// arguments that never passed the boundary; naming mrw's ack store as either
// quoted its JSON — checkpoint ids and all — back in the parse error. Both are
// refused, and neither answer carries the ids.
func TestTheStateCannotBeReadAsAListOrAPlan(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, ".st"))
	pending := filepath.Join(root, ".st", "mrw", "k", "pending.json")
	if err := os.MkdirAll(filepath.Dir(pending), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pending, []byte(`{"ck":"ck-SECRET"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := readIn(t, root, "--files-from", pending)
	if err == nil || !strings.Contains(errString(err), "own state") || strings.Contains(out+errString(err), "ck-SECRET") {
		t.Errorf("--files-from on the ack store: err %v\n%s", err, out)
	}
	wout, code := runIn(t, root, "write", "--no-check", pending)
	if code == 0 || !strings.Contains(wout, "own state") || strings.Contains(wout, "ck-SECRET") {
		t.Errorf("a plan read from the ack store: exit %d\n%s", code, wout)
	}
}
