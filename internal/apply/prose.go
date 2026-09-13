package apply

import (
	"path/filepath"
	"strings"
)

// proseExts is the ONE closed list ADR-054 names. Both arms read it: the CLI
// default check does not spawn on a plan whose written paths are all prose,
// and the balance delta is omitted on a prose hunk. Growing it is a new
// record, not an execute change — data files (`.jsonl`, `.yml`, `.json`) are
// deliberately absent because `.toml` must keep triggering and a blanket
// data exemption would take `Cargo.toml` with it.
var proseExts = map[string]bool{
	".md": true, ".markdown": true, ".txt": true, ".rst": true, ".adoc": true,
}

// IsProse reports whether a path's extension is on ADR-054's closed prose
// list. The comparison is on the lowercased extension; a path with no
// extension is not prose.
func IsProse(path string) bool {
	return proseExts[strings.ToLower(filepath.Ext(path))]
}
