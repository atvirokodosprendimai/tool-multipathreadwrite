package rooted

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// ADR-134. ADR-077 compared a served path, and each ancestor, with the state
// base, and never the served FILE with the files inside it: a hard link in the
// root to the ledger was read whole and matched by --grep. A file is mrw's
// state if it is the same file as one under the base, whatever it is called.

// errHardLinked is the refusal for a second name of a state file.
func errHardLinked(path string) error {
	return fmt.Errorf("%s is a hard link to a file in mrw's own state directory; mrw does not serve or edit its own ledger", path)
}

// linkedMaybe reports whether fi, a regular file at path, may have a second
// name: its link count is above one, or cannot be read. A file nothing else
// leads to is never compared, which is nearly every file; a pnpm store or a
// build cache is full of counts above one, and only identity with a state
// file refuses them.
func linkedMaybe(fi os.FileInfo, path string) bool {
	n, ok := linkCount(fi, path)
	return !ok || n > 1
}

// stateFiles are the regular files under the state base b that another name
// could lead to: those whose own link count is above one. Only they can have a
// second name, so a base of tens of thousands of directories yields a short
// list, usually none. It is read when a candidate needs it, never before.
func stateFiles(b string) []os.FileInfo {
	var out []os.FileInfo
	_ = filepath.WalkDir(b, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return nil //nolint:nilerr // a directory that cannot be listed, or a non-file, holds no name mrw can compare; the walk goes on
		}
		if fi, err := d.Info(); err == nil && linkedMaybe(fi, p) {
			out = append(out, fi)
		}
		return nil
	})
	return out
}

// sameAsAny reports whether fi is the same file as one of files.
func sameAsAny(fi os.FileInfo, files []os.FileInfo) bool {
	for _, f := range files {
		if os.SameFile(fi, f) {
			return true
		}
	}
	return false
}

// hardLinkedToState reports whether q, a path already resolved as far as it
// exists, is a regular file that is also a file under the state base.
func hardLinkedToState(q string) bool {
	fi, err := os.Lstat(q)
	if err != nil || !fi.Mode().IsRegular() || !linkedMaybe(fi, q) {
		return false
	}
	base, err := state.Base()
	if err != nil {
		return false
	}
	b, bi := resolvedBase(base)
	return bi != nil && sameAsAny(fi, stateFiles(b))
}
