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
