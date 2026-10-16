package state

import (
	"os"
	"path/filepath"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/links"
)

// Checkout is the git checkout that holds root (ADR-161 Decision 14): top is
// the nearest directory at or above root's resolved path with a .git entry — a
// directory, or a file as in a worktree or a submodule — and prefix is root's
// path below it. With no .git anywhere above, or with root itself holding it,
// top is root's resolved path and prefix is empty: the state root always had.
// The read ledger, the write lock, the write counter and the hard-link
// boundary key on top; everything else mrw keeps stays per root.
func Checkout(root string) (top, prefix string) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return root, ""
	}
	rp := links.Real(abs)
	for dir := rp; ; {
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			rel, err := filepath.Rel(dir, rp)
			if err != nil || rel == "." {
				return dir, ""
			}
			return dir, rel
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return rp, ""
		}
		dir = parent
	}
}
