package writer

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"path/filepath"
	"sort"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/regular"
)

// Snapshot is the hash of each file a write touched, taken just before its
// check runs (ADR-112): path to sha256, "" for a path that is not a regular
// file.
type Snapshot map[string]string

// Before hashes each file res wrote and did not remove, as it stands now —
// just before the check. The baseline is what is on disk, not the write's
// sha_after: a renamed relative symlink keeps the sha of the file it pointed at
// before the move and resolves to another after it, and a drift reported
// against that would be a lie (the Codex review of #309).
func Before(root string, res apply.Result) Snapshot {
	s := Snapshot{}
	for _, f := range res.Files {
		if !f.Written || f.Removed {
			continue
		}
		p := f.Path
		if f.RenamedTo != "" {
			p = f.RenamedTo
		}
		s[p] = hashOf(filepath.Join(root, filepath.FromSlash(p)))
	}
	return s
}

// Drift returns, sorted, each path in s whose bytes no longer hash as they did
// before the check — changed, gone, or no longer a regular file (ADR-112). A
// write's check runs after the write lock is released, so another writer, or
// the check itself, can change a file the write just landed, and the verdict
// is then about a tree that moved under it.
func Drift(root string, s Snapshot) []string {
	var out []string
	for p, sha := range s {
		if hashOf(filepath.Join(root, filepath.FromSlash(p))) != sha {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

// hashOf is the sha256 of full's bytes, streamed so a file that grew under the
// check costs no more memory than one that did not, or "" when full is not a
// regular file. It opens through regular.Open, so a file swapped for a FIFO is
// not waited on (ADR-109).
func hashOf(full string) string {
	f, fi, err := regular.Open(full)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if !fi.Mode().IsRegular() {
		return ""
	}
	if _, err := io.Copy(h, f); err != nil {
		return "unreadable: " + err.Error()
	}
	return hex.EncodeToString(h.Sum(nil))
}
