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

// win32Alias reports the first component of p that Windows does not keep as
// written, and the name left once the characters Windows drops are dropped.
//
// Win32 drops a trailing dot or space from a name, so "b.txt." and "b.txt "
// open b.txt, and it cannot create a name that ends in one; and a colon names
// an NTFS stream, so "b.txt::$DATA" opens b.txt's data. On Windows each let a
// write or an unlink land through a name that is not on disk while the receipt
// named the alias (ADR-071). "." and ".." end in a dot and mean what they say.
// Both separators are split because the function models Windows, wherever it
// is tested.
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

// win32Device reports p's last component when Go's rule for Windows device
// names says it may open one: CON, PRN, AUX, NUL, COM1–COM9 and LPT1–LPT9 (the
// digits ¹²³ too), CONIN$ and CONOUT$, in any case, with the name taken before
// its first dot or colon and trailing spaces dropped, as
// internal/filepathlite's isReservedName does. It is a candidate, not a
// verdict: whether "nul.txt" opens a device depends on the Windows version, so
// Resolve asks the OS (opensDevice) before it refuses. Only the last component
// is a candidate, because a device name in the middle of a path cannot be made
// as a directory and fails loudly on its own (ADR-076).
func win32Device(p string) string {
	p = p[len(filepath.VolumeName(p)):]
	parts := strings.FieldsFunc(p, func(r rune) bool { return r == '/' || r == '\\' })
	if len(parts) == 0 {
		return ""
	}
	last := parts[len(parts)-1]
	base := last
	if i := strings.IndexAny(base, ".:"); i >= 0 {
		base = base[:i]
	}
	base = strings.TrimRight(base, " ")
	if isReservedBase(base) {
		return last
	}
	return ""
}

// isReservedBase is Go's isReservedBaseName: the device names themselves. The
// case is folded in ASCII only, byte for byte: strings.ToUpper maps "ı" to "I",
// which shortened "ıı" from four bytes to two, and slicing the result by the
// original length panicked on a filename (Codex review of #237).
func isReservedBase(name string) bool {
	upper := asciiUpper(name)
	switch upper {
	case "CON", "PRN", "AUX", "NUL", "CONIN$", "CONOUT$":
		return true
	}
	if len(name) < 4 || (upper[:3] != "COM" && upper[:3] != "LPT") {
		return false
	}
	switch rest := name[3:]; rest {
	case "\u00b9", "\u00b2", "\u00b3":
		return true
	default:
		return len(rest) == 1 && rest[0] >= '1' && rest[0] <= '9'
	}
}

// asciiUpper upper-cases a-z and leaves every other byte as it is, so the
// result is exactly as long as s.
func asciiUpper(s string) string {
	b := []byte(s)
	for i, c := range b {
		if 'a' <= c && c <= 'z' {
			b[i] = c - ('a' - 'A')
		}
	}
	return string(b)
}
