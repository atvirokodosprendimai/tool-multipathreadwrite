package rooted

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// The Codex review of #361, second pass. With this checkout's state directory
// moved and replaced by a link to it, the listing saw a single non-file entry,
// came back empty and complete, and the live ledger was served by its alias.
// The directory is listed where it really is.
func TestAHardLinkIsRefusedWhenTheStateDirectoryIsItselfALink(t *testing.T) {
	root, st := t.TempDir(), t.TempDir()
	t.Setenv("XDG_STATE_HOME", st)
	dir, err := state.Dir(root)
	if err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(st, "moved")
	if err := os.Rename(dir, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(moved, dir); err != nil {
		t.Skipf("links are not available here: %v", err)
	}
	ledger := filepath.Join(moved, "seen")
	if err := os.WriteFile(ledger, []byte("#mrw-seen v4\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(ledger, filepath.Join(root, "hl.txt")); err != nil {
		t.Skipf("hard links are not available here: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "ord.txt"), []byte("ordinary\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(filepath.Join(root, "ord.txt"), filepath.Join(root, "ord2.txt")); err != nil {
		t.Skipf("hard links are not available here: %v", err)
	}
	r := NewResolver(root)
	for who, resolve := range map[string]func(string) (string, error){
		"Resolve":          func(p string) (string, error) { return Resolve(root, p) },
		"Resolver.Resolve": r.Resolve,
	} {
		if _, err := resolve("hl.txt"); err == nil || !strings.Contains(err.Error(), "own state") {
			t.Errorf("%s(hl.txt) with a linked state directory = %v, want it refused as mrw's own state", who, err)
		}
		// Listed where it really is, the directory tells the ledger's alias from
		// an ordinary hard link; listed as spelled it could only refuse both.
		if _, err := resolve("ord2.txt"); err != nil {
			t.Errorf("%s(ord2.txt), an ordinary hard link, was refused with a linked state directory: %v", who, err)
		}
	}
}
