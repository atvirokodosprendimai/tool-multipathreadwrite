package writer

import (
	"io"
	"path/filepath"
	"sort"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/regular"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// Drift returns, sorted, each file res wrote whose bytes no longer hash to the
// sha_after the write gave it, or that is gone or no longer a regular file
// (ADR-112). A write's check runs after the write lock is released, so another
// writer — or the check itself, which may format or generate files — can
// change a file the write just landed; the check's verdict is then about a tree
// that is not the write's. It looks where the bytes went: a rename's
// destination, a symlink's target. A removed path, and a file with no
// sha_after, has nothing to compare.
func Drift(root string, res apply.Result) []string {
	var out []string
	for _, f := range res.Files {
		if !f.Written || f.Removed || f.SHAAfter == "" {
			continue
		}
		p := f.Path
		if f.RenamedTo != "" {
			p = f.RenamedTo
		}
		if f.Target != "" {
			p = f.Target
		}
		if !holds(filepath.Join(root, filepath.FromSlash(p)), f.SHAAfter) {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

// holds reports whether full is still a regular file whose bytes hash to sha.
// It opens through regular.Open, so a file swapped for a FIFO is drift rather
// than a read that waits (ADR-109).
func holds(full, sha string) bool {
	f, fi, err := regular.Open(full)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	if !fi.Mode().IsRegular() {
		return false
	}
	b, err := io.ReadAll(f)
	return err == nil && seen.SHA(b) == sha
}
