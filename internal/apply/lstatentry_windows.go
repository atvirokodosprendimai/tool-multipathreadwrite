//go:build windows

package apply

import (
	"os"
	"path/filepath"
	"strings"
)

// lstatEntry is os.Lstat for an entry the directory lists. Win32 strips a
// trailing dot or space from a name, so asked plainly a listed `dot.` is "not
// found", and a listed `plain.txt.` opens plain.txt itself, which respelling
// would count as a second link to the source. Such a name is asked through an
// extended-length path, which Win32 does not normalise, so every listed entry
// is the entry it names (ADR-129 amendment; the Codex review of #360 and the
// review of the fix). An entry not found either way has gone.
func lstatEntry(p string) (os.FileInfo, error) {
	base := filepath.Base(p)
	edge := strings.HasSuffix(base, ".") || strings.HasSuffix(base, " ")
	if edge && filepath.IsAbs(p) && !strings.HasPrefix(p, `\\`) {
		return os.Lstat(`\\?\` + p)
	}
	return os.Lstat(p)
}
