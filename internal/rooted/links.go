package rooted

import (
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/links"
)

// The walk, its filesystem seam and IsRooted live in internal/links, which
// internal/state imports too, so the boundary and the state key canonicalise a
// root one way (ADR-103). These keep rooted's callers as they were.
const followLinks = links.Follow

var osLinks = links.OS

func throughLinks(p string, lfs links.FS) (string, error) { return links.Through(p, lfs) }

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
		// ADR-086: Windows converts a path to UTF-16 and maps a byte that is
		// not valid UTF-8 to U+FFFD without an error, so the name on disk
		// would be another one than the plan's.
		if !utf8.ValidString(c) {
			return c, strings.ToValidUTF8(c, string(utf8.RuneError))
		}
	}
	return "", ""
}

// win32Device reports the first component of p that Go's rule for Windows
// device names says may open one: CON, PRN, AUX, NUL, COM1–COM9 and LPT1–LPT9
// (the digits ¹²³ too), CONIN$ and CONOUT$, in any case, with the name taken
// before its first dot or colon and trailing spaces dropped, as
// internal/filepathlite's isReservedName does. On Windows Resolve refuses every
// one by name: whether the OS opens "nul.txt" as a device differs by build and
// by API, and asking one API made files another could not reach (ADR-081).
// Every component counts, not only the last: on a build that makes a file
// `con`, a directory `con` is made the same way (the review of #243).
func win32Device(p string) string {
	p = p[len(filepath.VolumeName(p)):]
	for _, part := range strings.FieldsFunc(p, func(r rune) bool { return r == '/' || r == '\\' }) {
		base := part
		if i := strings.IndexAny(base, ".:"); i >= 0 {
			base = base[:i]
		}
		if isReservedBase(strings.TrimRight(base, " ")) {
			return part
		}
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
