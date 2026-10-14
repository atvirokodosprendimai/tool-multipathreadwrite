package rooted

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The Codex review of #361, third pass. With the ledger moved out of the state
// directory and a link left in its place, the live file was reached by a link
// the listing skipped, and its hard-linked alias in the checkout was served.
// A link in the state directory is followed to the file it leads to.
func TestAHardLinkToALedgerTheStateDirectoryLinksToIsRefused(t *testing.T) {
	root, ledger := hardLinkRoot(t)
	moved := filepath.Join(t.TempDir(), "moved-seen")
	if err := os.Rename(ledger, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(moved, ledger); err != nil {
		t.Skipf("links are not available here: %v", err)
	}
	if err := os.Remove(filepath.Join(root, "hl.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(moved, filepath.Join(root, "hl.txt")); err != nil {
		t.Skipf("hard links are not available here: %v", err)
	}
	r := NewResolver(root)
	for who, resolve := range map[string]func(string) (string, error){
		"Resolve":          func(p string) (string, error) { return Resolve(root, p) },
		"Resolver.Resolve": r.Resolve,
	} {
		if _, err := resolve("hl.txt"); err == nil || !strings.Contains(err.Error(), "own state") {
			t.Errorf("%s(hl.txt), an alias of a ledger the state directory links to = %v, want it refused as mrw's own state", who, err)
		}
	}
}
