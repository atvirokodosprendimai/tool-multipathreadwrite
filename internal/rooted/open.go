package rooted

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/regular"
)

// openGapFn runs between the last name-based step of OpenRead and the open
// beneath the held root. It is a seam for a test to swap a directory for a link
// there (#727).
var openGapFn = func() {}

// OpenRead opens for reading the file full, which Resolve returned for a path
// under root, and returns it with its descriptor's FileInfo (regular.OpenIn).
//
// Resolve decides by name that the path is inside the root, and a later open by
// name can lead somewhere else if a directory was swapped for a link in
// between (#727, ADR-007). So the open is made through an os.Root held on the
// root, which resolves every component below the handle and refuses one that
// leaves it. The path is handed over RESOLVED as far as it exists, as apply's
// tree does (ADR-106): os.Root follows a relative link that stays inside but
// refuses an absolute one, and a link inside the root, or a Windows junction,
// must keep reading (ADR-071). A path that resolves outside now is refused here
// with Resolve's own error.
func OpenRead(root, full string) (*os.File, fs.FileInfo, error) {
	absRoot, err := Abs(root)
	if err != nil {
		return nil, nil, err
	}
	resolved := RealAsFarAsItExists(full)
	rel, err := filepath.Rel(absRoot, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return nil, nil, &OutsideError{Path: full, Resolved: resolved, Root: absRoot, Exists: true}
	}
	r, err := os.OpenRoot(absRoot)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = r.Close() }()
	openGapFn()
	return regular.OpenIn(r, rel)
}
