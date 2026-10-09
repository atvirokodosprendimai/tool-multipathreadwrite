//go:build windows

package apply

import (
	"os"
	"path/filepath"
	"strings"
)

// extendedPath is p spelled as an extended-length path, which Win32 does not
// normalise: a drive path gains `\\?\`, an ordinary UNC path `\\server\share\x`
// becomes `\\?\UNC\server\share\x`, a device-form drive path `\\.\C:\x` becomes
// `\\?\C:\x` (the `\\.\` form still strips a trailing dot), and a path already
// extended is kept. A relative path, or a device that is not a drive, gives "".
func extendedPath(p string) string {
	switch {
	case strings.HasPrefix(p, `\\?\`):
		return p
	case strings.HasPrefix(p, `\\.\`):
		if len(p) > 6 && p[5] == ':' {
			return `\\?\` + p[4:]
		}
		return ""
	case strings.HasPrefix(p, `\\`):
		return `\\?\UNC\` + p[2:]
	case filepath.IsAbs(p):
		return `\\?\` + p
	}
	return ""
}

// lstatEntry is os.Lstat for an entry the directory lists. Win32 strips a
// trailing dot or space from a name, so asked plainly a listed `dot.` is "not
// found", and a listed `plain.txt.` opens plain.txt itself, which respelling
// would count as a second link to the source. Such a name is asked through its
// extended-length path, so every listed entry is the entry it names (ADR-129
// amendment; the Codex reviews of #360 and the review of the fix). An entry not
// found either way has gone.
func lstatEntry(p string) (os.FileInfo, error) {
	base := filepath.Base(p)
	if strings.HasSuffix(base, ".") || strings.HasSuffix(base, " ") {
		if ep := extendedPath(p); ep != "" {
			return os.Lstat(ep)
		}
	}
	return os.Lstat(p)
}
