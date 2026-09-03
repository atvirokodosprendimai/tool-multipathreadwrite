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

// Abs is the root as the boundary compares it: absolute, and with symlinks
// resolved so a root reached through one — /var on macOS — is the same string
// as the paths it is compared against.
//
// Exported because every caller that pre-screens a path needs the identical
// root, and three hand-rolled copies of these four lines is how the two
// spellings drift apart.
func Abs(root string) (string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if real, err := filepath.EvalSymlinks(absRoot); err == nil {
		absRoot = real
	}
	return absRoot, nil
}
func Resolve(root, path string) (string, error) {
	absRoot, err := Abs(root)
	if err != nil {
		return "", err
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

// IsRooted reports whether p names a location of its own, rather than one to be
// resolved relative to a root.
//
// filepath.IsAbs is not enough, and the gap is Windows-only. There, `\etc\hosts`
// and `/etc/hosts` are ROOTED — they name the root of the current drive — but
// IsAbs is FALSE for both, because neither carries a volume. `C:etc` is
// drive-relative and equally not root-relative, and IsAbs is false for that too.
// So every caller asking "did the caller hand me a path that is not relative to
// my root?" answered no on Windows and joined it on.
//
// In read and apply that only misdirects a message. In check it is a silent
// PASS: the joined path places no package, the run falls back to the full check
// and the verdict is green, which is the failure class this project exists to
// refuse. Found by the windows CI job on its first run, 2026-09-03.
//
// A leading backslash is rooted on Windows and an ordinary filename character on
// POSIX, so it counts only where it means something.
func IsRooted(p string) bool {
	if p == "" {
		return false
	}
	if filepath.IsAbs(p) || filepath.VolumeName(p) != "" {
		return true
	}
	return p[0] == '/' || (filepath.Separator == '\\' && p[0] == '\\')
}
