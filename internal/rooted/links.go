package rooted

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// linkFS is what the link walk asks of a filesystem. It is a value rather than
// the os package so the walk Windows needs can be driven, in a test, on a
// platform that has no junctions (ADR-071).
type linkFS struct {
	lstat    func(string) (os.FileInfo, error)
	readlink func(string) (string, error)
}

// osLinks is the real filesystem.
var osLinks = linkFS{lstat: os.Lstat, readlink: os.Readlink}

// maxLinks bounds a walk through links that point at each other.
const maxLinks = 255

// throughLinks returns p with every existing component that is a symlink or a
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
func throughLinks(p string, lfs linkFS) (string, error) {
	p = filepath.Clean(p)
	vol := filepath.VolumeName(p)
	rest := components(p[len(vol):])
	cur := vol + string(filepath.Separator)
	hops := 0
	for len(rest) > 0 {
		next := filepath.Join(cur, rest[0])
		fi, err := lfs.lstat(next)
		if err != nil {
			return filepath.Join(append([]string{next}, rest[1:]...)...), nil
		}
		rest = rest[1:]
		if fi.Mode()&(os.ModeSymlink|os.ModeIrregular) == 0 {
			cur = next
			continue
		}
		target, err := lfs.readlink(next)
		if err != nil {
			if fi.Mode()&os.ModeSymlink == 0 && errors.Is(err, fs.ErrNotExist) {
				cur = next // a reparse point that redirects nothing
				continue
			}
			return "", fmt.Errorf("%s is a link or junction mrw cannot follow: %w", next, err)
		}
		if hops++; hops > maxLinks {
			return "", fmt.Errorf("%s leads through more than %d links", p, maxLinks)
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
	return strings.FieldsFunc(p, func(r rune) bool { return os.IsPathSeparator(uint8(r)) })
}

// win32Alias reports the first component of p that Windows reads as a
// different name, and the name it reads.
//
// Win32 strips a trailing dot or space from every component, so "b.txt." and
// "b.txt " open b.txt; and a colon names an NTFS stream, so "b.txt::$DATA"
// opens b.txt's data. On Windows each let a write or an unlink land through a
// name that is not on disk while the receipt named the alias (ADR-071). "." and
// ".." end in a dot and mean what they say. Both separators are split because
// the function models Windows, wherever it is tested.
func win32Alias(p string) (comp, reads string) {
	p = p[len(filepath.VolumeName(p)):]
	for _, c := range strings.FieldsFunc(p, func(r rune) bool { return r == '/' || r == '\\' }) {
		if c == "." || c == ".." {
			continue
		}
		if i := strings.IndexByte(c, ':'); i >= 0 {
			return c, c[:i]
		}
		if t := strings.TrimRight(c, ". "); t != c {
			return c, t
		}
	}
	return "", ""
}
