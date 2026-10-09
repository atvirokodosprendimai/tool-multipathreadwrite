package rooted

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// hardLinkRoot is a root holding hl.txt, a hard link to a file in mrw's state
// base (outside the root), and ord2.txt, a hard link to an ordinary file.
func hardLinkRoot(t *testing.T) (root, ledger string) {
	t.Helper()
	root, st := t.TempDir(), t.TempDir()
	t.Setenv("XDG_STATE_HOME", st)
	dir, err := state.Dir(root)
	if err != nil {
		t.Fatal(err)
	}
	ledger = filepath.Join(dir, "seen")
	for p, body := range map[string]string{ledger: "#mrw-seen v4\n", filepath.Join(root, "ord.txt"): "ordinary\n"} {
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for from, to := range map[string]string{ledger: "hl.txt", filepath.Join(root, "ord.txt"): "ord2.txt"} {
		if err := os.Link(from, filepath.Join(root, to)); err != nil {
			t.Skipf("hard links are not available here: %v", err)
		}
	}
	return root, ledger
}

// ADR-134. ADR-077 promises mrw's own state is never served, and compared a
// path with the state base but never the served FILE with the files inside it:
// a hard link in the root to the ledger was read whole and matched by --grep
// (the Windows chaos round on v1.52.0, reproduced on macOS). A hard link to an
// ordinary file is served; a pnpm store is full of them.
func TestAHardLinkToMrwsStateIsRefused(t *testing.T) {
	root, _ := hardLinkRoot(t)
	r := NewResolver(root)
	for who, resolve := range map[string]func(string) (string, error){
		"Resolve":          func(p string) (string, error) { return Resolve(root, p) },
		"Resolver.Resolve": r.Resolve,
	} {
		if _, err := resolve("hl.txt"); err == nil || !strings.Contains(err.Error(), "own state") {
			t.Errorf("%s(hl.txt), a hard link to the ledger = %v, want it refused as mrw's own state", who, err)
		}
		for _, p := range []string{"ord.txt", "ord2.txt"} {
			if _, err := resolve(p); err != nil {
				t.Errorf("%s(%s) refused a file that is not mrw's state: %v", who, p, err)
			}
		}
	}
}

// A symlink to the hard link is judged by the file it leads to.
func TestASymlinkToAHardLinkToMrwsStateIsRefused(t *testing.T) {
	root, _ := hardLinkRoot(t)
	if err := os.Symlink("hl.txt", filepath.Join(root, "sl.txt")); err != nil {
		t.Skipf("symlinks are not available here: %v", err)
	}
	if _, err := Resolve(root, "sl.txt"); err == nil || !strings.Contains(err.Error(), "own state") {
		t.Errorf("Resolve(sl.txt) = %v, want it refused as mrw's own state", err)
	}
}

// The ledger is saved by rename, so a link made before the save is afterwards a
// copy of a past ledger, the file of nothing mrw keeps (the record's Out of
// Scope): it is served, and the test pins that it is a decision.
func TestALinkTheNextSaveReplacedIsAnOrdinaryCopy(t *testing.T) {
	root, ledger := hardLinkRoot(t)
	next := ledger + ".tmp"
	if err := os.WriteFile(next, []byte("#mrw-seen v4\nnewer\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(next, ledger); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(root, "hl.txt"); err != nil {
		t.Errorf("Resolve(hl.txt) after the ledger was saved by rename = %v, want the stale copy served", err)
	}
}
