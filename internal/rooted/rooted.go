// Package rooted turns a caller-supplied path into the file it names inside a
// root, and refuses one that leaves.
//
// It exists because the boundary was implemented once, on the write path, and
// therefore held on exactly one of the two ways into the tree: `mrw -C repo
// read ../outside.txt` served the file at exit 0, and a symlink out of the tree
// was followed, while the identical path in a write plan was refused by name.
//
// A boundary that holds on one path and not the other is not a boundary, it is
// a coincidence of which function you happened to call — so there is one
// implementation and both callers use it.
package rooted

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Resolve returns the absolute file that path names under root, and an error if
// it resolves outside it.
//
// Symlinks are resolved before the check, because following one out of the tree
// is the same escape wearing a different hat. A path that does not exist yet is
// checked lexically — `create` is entitled to name a file that is not there,
// and filepath.Join has already cleaned it.
func Resolve(root, path string) (string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if real, err := filepath.EvalSymlinks(absRoot); err == nil {
		absRoot = real
	}

	full := filepath.Join(absRoot, path)
	check := full
	if real, err := filepath.EvalSymlinks(full); err == nil {
		check = real
	}
	if !Contains(absRoot, check) {
		return "", fmt.Errorf("%s resolves to %s, which is outside the root %s", path, check, absRoot)
	}
	return full, nil
}

// Contains reports whether p is absRoot itself or something beneath it. The
// separator matters: without it, "/repo-backup" counts as inside "/repo".
func Contains(absRoot, p string) bool {
	return p == absRoot || strings.HasPrefix(p, absRoot+string(filepath.Separator))
}

// Descendable reports whether a walk may go INTO path, an entry found inside
// absRoot. absRoot must already be canonical — Resolve produces it, and a walk
// resolves its root once before it starts.
//
// A symlinked DIRECTORY is refused, by rule 3 of ADR-007 and by that rule
// alone: following one can leave the tree and can loop, and a link pointing at
// its own parent is a walk that never ends. Nothing is lost by refusing —
// whatever it points at inside the root is reached by its real name anyway,
// and a symlinked FILE is still served, because that question is asked after
// Resolve rather than here.
//
// Anything that is not a directory answers false with no error: a regular file
// is not this function's question, and saying yes would send a walk into
// something that cannot be walked.
func Descendable(absRoot, path string) (bool, error) {
	// Ask the two questions separately, and in this order, so that rule 3 is
	// the thing that refuses a symlinked directory rather than an accident of
	// Lstat's semantics. Lstat reports a symlink's own mode, whose IsDir is
	// FALSE even when the target is a directory — so a single Lstat would
	// refuse the link on "not a directory" and the rule would be unreachable
	// dead code that no test could distinguish from the rule working.
	lst, err := os.Lstat(path)
	if err != nil {
		return false, err
	}
	st, err := os.Stat(path)
	if err != nil {
		// A broken link, or an entry that vanished mid-walk. Not descendable,
		// and not an error worth aborting a batch for.
		return false, nil
	}
	if !st.IsDir() {
		return false, nil
	}
	if lst.Mode()&os.ModeSymlink != 0 {
		return false, nil
	}
	// Safe to resolve now: the entry itself is a real directory, so this
	// canonicalises the path it was reached by rather than following it
	// somewhere new. It matters because a caller's path may run through a
	// symlinked ancestor — /var on macOS — and the comparison below is against
	// a canonical root.
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false, err
	}
	return Contains(absRoot, real), nil
}
