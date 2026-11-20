package writer

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/regular"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted"
)

// Snapshot is the hash of each file a write touched, taken just before its
// check runs (ADR-112): path to sha256, "" for a path that is not a regular
// file.
type Snapshot map[string]string

// Before hashes each file res wrote and did not remove, as it stands now —
// just before the check. The baseline is what is on disk, not the write's
// sha_after: it is the bytes the check will see, whatever the receipt said
// (the Codex review of #309; since #712 a moved link's sha_after is what it
// reads at its new place).
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
		// A moved link is hashed only where the boundary allows it: a target
		// outside the root or in mrw's state is never read (#712, Codex on
		// #725); it can still drift inside the root (the Codex review of #309).
		if _, err := rooted.Resolve(root, p); err != nil {
			continue
		}
		s[p] = hashIn(root, p)
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
		// Judged again: a path that now resolves outside the root has drifted,
		// and is not read.
		if _, err := rooted.Resolve(root, p); err != nil || hashIn(root, p) != sha {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

// hashIn is the sha256 of the bytes of p, root-relative, streamed so a file
// that grew under the check costs no more memory than one that did not, or ""
// when p is not a regular file. It opens through an os.Root held on root, so
// a link or a parent swapped after the boundary was judged cannot lead the
// open outside it (Codex on #725), and through regular.OpenIn, so a file
// swapped for a FIFO is not waited on (ADR-109).
func hashIn(root, p string) string {
	r, err := os.OpenRoot(root)
	if err != nil {
		return ""
	}
	defer func() { _ = r.Close() }()
	f, fi, err := regular.OpenIn(r, filepath.FromSlash(p))
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
