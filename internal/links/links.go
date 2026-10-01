// Package links follows symlinks and Windows junctions the one way mrw's
// boundary and its state key both need (ADR-071, ADR-103).
//
// It is a leaf because both of its callers need it and one imports the other:
// internal/rooted (the boundary) imports internal/state (the per-checkout key).
// While the walk lived in rooted, state resolved a root with EvalSymlinks alone,
// which stops at a junction — so a checkout reached through one got its own
// state directory and its own writer lock.
package links

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// FS is what the link walk asks of a filesystem. It is a value rather than
// the os package so the walk Windows needs can be driven, in a test, on a
// platform that has no junctions (ADR-071).
type FS struct {
	Lstat    func(string) (os.FileInfo, error)
	Readlink func(string) (string, error)
}

// OS is the real filesystem.
var OS = FS{Lstat: os.Lstat, Readlink: os.Readlink}

// MaxLinks bounds a walk through links that point at each other.
const MaxLinks = 255

// Through returns p with every existing component that is a symlink or a
// junction replaced by its target, so the boundary compares where p really
// leads rather than how it is spelled.
//
// It exists because of Go 1.23. Since then a Windows junction is reported as
// ModeIrregular rather than ModeSymlink and filepath.EvalSymlinks no longer
// follows it, so Resolve met ENOTDIR at the junction, took it for a missing
// leaf, and judged the path by a lexical ancestor that was inside the root.
// Three Windows sessions read, wrote, created, renamed and unlinked through a
// junction out of the root at exit 0.
//
// A component that does not exist ends the walk and the rest is kept as
// written: a create names a file that is not there yet. An irregular entry
// whose Readlink answers ENOENT — Go's answer for a reparse point that is
// neither a symlink nor a junction, such as a OneDrive placeholder — redirects
// nothing and is kept. Any other Readlink failure refuses: a link mrw cannot
// read is a link whose destination it does not know.
func Through(p string, lfs FS) (string, error) {
	p = filepath.Clean(p)
	vol := filepath.VolumeName(p)
	rest := components(p[len(vol):])
	cur := vol + string(filepath.Separator)
	hops := 0
	for len(rest) > 0 {
		next := filepath.Join(cur, rest[0])
		fi, err := lfs.Lstat(next)
		if err != nil {
			// A component that is not there, or cannot be (an invalid name, a
			// glob a shell left unexpanded), ends the walk: a create names a
			// file that does not exist yet, and a name that cannot exist cannot
			// be a link. One that exists but may not be examined is refused: it
			// is not knowledge of where the path leads (review of #228, A1).
			if !errors.Is(err, fs.ErrPermission) {
				return filepath.Join(append([]string{next}, rest[1:]...)...), nil
			}
			return "", fmt.Errorf("%s cannot be examined, so mrw cannot tell where it leads: %w", next, err)
		}
		rest = rest[1:]
		if fi.Mode()&(os.ModeSymlink|os.ModeIrregular) == 0 {
			cur = next
			continue
		}
		target, err := lfs.Readlink(next)
		if err != nil {
			if fi.Mode()&os.ModeSymlink == 0 && errors.Is(err, fs.ErrNotExist) {
				cur = next // a reparse point that redirects nothing
				continue
			}
			return "", fmt.Errorf("%s is a link or junction mrw cannot follow: %w", next, err)
		}
		if hops++; hops > MaxLinks {
			return "", fmt.Errorf("%s leads through more than %d links", p, MaxLinks)
		}
		// A relative target is relative to the directory holding the link; a
		// rooted one without a volume is on the link's own drive.
		if !IsRooted(target) {
			target = filepath.Join(cur, target)
		} else if filepath.VolumeName(target) == "" {
			target = vol + target
		}
		target = filepath.Clean(target)
		vol = filepath.VolumeName(target)
		rest = append(components(target[len(vol):]), rest...)
		cur = vol + string(filepath.Separator)
	}
	return cur, nil
}

// components splits a volume-less path at this platform's separators.
func components(p string) []string {
	// ADR-108: a separator is an ASCII rune the OS calls one. uint8(r) narrowed
	// U+042F (Я) to '/' and U+015C (Ŝ) to '\', splitting valid names.
	return strings.FieldsFunc(p, func(r rune) bool { return r < utf8.RuneSelf && os.IsPathSeparator(uint8(r)) })
}

// IsRooted reports whether p names a location of its own, rather than one to be
// resolved relative to a root.
//
// filepath.IsAbs is not enough, and the gap is Windows-only. There, `\etc\hosts`
// and `/etc/hosts` are ROOTED — they name the root of the current drive — but
// IsAbs is FALSE for both, because neither carries a volume. `C:etc` is
// drive-relative and equally not root-relative, and IsAbs is false for that too.
// A leading backslash is rooted on Windows and an ordinary filename character on
// POSIX, so it counts only where it means something. rooted.IsRooted is this
// function; the walk needs it, so it lives here.
func IsRooted(p string) bool {
	if p == "" {
		return false
	}
	if filepath.IsAbs(p) || filepath.VolumeName(p) != "" {
		return true
	}
	return p[0] == '/' || (filepath.Separator == '\\' && p[0] == '\\')
}

// Real is p cleaned, with its links resolved — through junctions too where
// Follow is on — or p cleaned when they cannot be. It is the one
// canonicalisation of a root: the boundary compares paths against it, and the
// state directory is keyed by it (ADR-103).
func Real(p string) string { return realVia(p, OS, Follow) }

// realVia is Real over lfs, with the walk on or off, so a test can drive the
// junction case on a platform that has none.
func realVia(p string, lfs FS, follow bool) string {
	p = filepath.Clean(p)
	if follow {
		if t, err := Through(p, lfs); err == nil {
			p = t
		}
	}
	if real, err := filepath.EvalSymlinks(p); err == nil {
		return real
	}
	return p
}
