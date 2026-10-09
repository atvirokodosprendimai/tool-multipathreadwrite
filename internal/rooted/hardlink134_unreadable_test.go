//go:build unix

package rooted

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The Codex review of #361, fourth pass. A link in the state directory whose
// target cannot be followed (the directory on its way has no search permission)
// was passed over as if it led nowhere, and the target's readable alias in the
// checkout was served. The comparison is incomplete, so a file with a second
// name is refused.
func TestAnUnfollowableLinkInTheStateDirectoryRefusesAFileWithASecondName(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root follows any path")
	}
	root, ledger := hardLinkRoot(t)
	hidden := filepath.Join(t.TempDir(), "hidden")
	if err := os.MkdirAll(hidden, 0o700); err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(hidden, "seen")
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
	if err := os.Chmod(hidden, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(hidden, 0o700) })
	if _, err := Resolve(root, "hl.txt"); err == nil || !strings.Contains(err.Error(), "cannot tell") {
		t.Errorf("Resolve(hl.txt), a link whose state-directory name cannot be followed = %v, want it refused as not comparable", err)
	}
}
