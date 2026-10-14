package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// The Codex review of #361, fourth pass. A plan file and a --files-from list
// are shell arguments that never pass rooted.Resolve; ADR-077 refused them by
// spelling, so a hard link to the ack store was opened and its JSON, checkpoint
// ids and all, quoted back in the parse error. Both are refused by identity.
func TestAHardLinkToTheStateCannotBeReadAsAListOrAPlan(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	dir, err := state.Dir(root)
	if err != nil {
		t.Fatal(err)
	}
	pending := filepath.Join(dir, "pending.json")
	if err := os.WriteFile(pending, []byte(`{"ck":"ck-SECRET"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "hl.txt")
	if err := os.Link(pending, alias); err != nil {
		t.Skipf("hard links are not available here: %v", err)
	}
	out, err := readIn(t, root, "--files-from", alias)
	if err == nil || !strings.Contains(errString(err), "own state") || strings.Contains(out+errString(err), "ck-SECRET") {
		t.Errorf("--files-from on a hard link to the ack store: err %v\n%s", err, out)
	}
	wout, code := runIn(t, root, "write", "--no-check", alias)
	if code == 0 || !strings.Contains(wout, "own state") || strings.Contains(wout, "ck-SECRET") {
		t.Errorf("a plan read from a hard link to the ack store: exit %d\n%s", code, wout)
	}
}
