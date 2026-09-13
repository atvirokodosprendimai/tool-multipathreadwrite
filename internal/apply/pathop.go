package apply

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted"
)

func isPathOp(op string) bool {
	return op == "unlink" || op == "rename"
}

// nestedPath reports whether a and b name the same location or one sits
// under the other. Exact-string dest collisions have their own messages;
// this is the ancestor/descendant hole those miss: create new/child.txt
// then rename onto new validates, content commits first, and the rename
// then fails after the tree has already changed.
func nestedPath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if a == "." || b == "." {
		return a == b
	}
	if a == b {
		return true
	}
	sep := string(filepath.Separator)
	return strings.HasPrefix(a+sep, b+sep) || strings.HasPrefix(b+sep, a+sep)
}

// planPathOp validates a single unlink or rename hunk. The caller has already
// refused mixing those ops with line-edits on the same path.
func planPathOp(root, path, full string, h hunk, orig []string, existed bool, shaBefore string, unlinked, produced map[string]bool, destCount map[string]int, covered func(hunk, int, int) bool, fail func(hunk, string, ...any), out map[int]HunkResult) bool {
	if h.StartPat != nil || h.Start != 0 || h.End != 0 || h.RelEnd > 0 {
		fail(h, "%s takes no address, use %q", h.Op, "-")
		return false
	}
	if h.Anchor != "" || h.Lines >= 0 {
		fail(h, "%s takes no anchor= or lines=", h.Op)
		return false
	}
	if h.Op == "unlink" && len(h.Body) != 0 {
		fail(h, "unlink takes no body")
		return false
	}
	if h.Op == "rename" && (len(h.Body) != 1 || strings.TrimSpace(h.Body[0]) == "") {
		fail(h, "rename body is the dest path: one line")
		return false
	}
	if h.SHA != "" {
		switch {
		case !existed:
			fail(h, "sha=%s given but %s does not exist", h.SHA, path)
			return false
		case !strings.HasPrefix(shaBefore, h.SHA):
			fail(h, "file changed: sha is %s, plan expected %s", shaShown(shaBefore, h.SHA), h.SHA)
			return false
		}
	}
	if !existed {
		if rooted.IsRooted(h.Path) {
			fail(h, "%s is absolute, and every path in a plan is relative to the root: mrw looked for %s, "+
				"which does not exist", h.Path, full)
			return false
		}
		fail(h, "%s does not exist", path)
		return false
	}
	total := len(orig)
	if total > 0 && !covered(h, 1, total) {
		return false
	}
	if h.Op == "rename" {
		raw := strings.TrimSpace(h.Body[0])
		if rooted.IsRooted(raw) {
			fail(h, "%s is absolute, and every path in a plan is relative to the root: mrw looked for %s, "+
				"which does not exist", raw, raw)
			return false
		}
		dest := filepath.Clean(raw)
		if dest == path || dest == "." {
			fail(h, "rename dest %s is the source", dest)
			return false
		}
		if destCount[dest] > 1 {
			fail(h, "rename dest %s is named by more than one hunk", dest)
			return false
		}
		if produced[dest] {
			fail(h, "rename dest %s is also written by another hunk in this plan", dest)
			return false
		}
		for other := range destCount {
			if other != dest && nestedPath(dest, other) {
				fail(h, "rename dest %s nests with rename dest %s", dest, other)
				return false
			}
		}
		for p := range produced {
			if nestedPath(dest, p) {
				fail(h, "rename dest %s nests with %s written by another hunk in this plan", dest, p)
				return false
			}
		}
		destFull, err := resolve(root, dest)
		if err != nil {
			fail(h, "%s", err.Error())
			return false
		}
		if _, err := os.Lstat(destFull); err == nil && !unlinked[dest] {
			fail(h, "rename dest %s already exists — unlink it in this plan, or pick another path", dest)
			return false
		}
	}
	out[h.Index] = HunkResult{
		Path: path, Addr: "-", Op: h.SrcOp, SrcLine: h.SrcLine, Status: StatusOK,
	}
	return true
}

type aside struct {
	tmp  string
	orig string
}

func commitPathOps(res *Result, pathOps []pending) error {
	var asides []aside
	restore := func() {
		for i := len(asides) - 1; i >= 0; i-- {
			_ = os.Rename(asides[i].tmp, asides[i].orig)
		}
	}
	unlinkOne := func(w pending) error {
		dir := filepath.Dir(w.full)
		tmp, err := os.CreateTemp(dir, ".mrw-aside-*")
		if err != nil {
			return err
		}
		name := tmp.Name()
		if err := tmp.Close(); err != nil {
			os.Remove(name)
			return err
		}
		if err := os.Remove(name); err != nil {
			return err
		}
		if err := os.Rename(w.full, name); err != nil {
			return err
		}
		asides = append(asides, aside{tmp: name, orig: w.full})
		w.file.Written = true
		w.file.Removed = true
		w.file.LinesTo = 0
		w.file.SHAAfter = ""
		res.Files = append(res.Files, w.file)
		return nil
	}
	renameOne := func(w pending) error {
		if err := os.MkdirAll(filepath.Dir(w.renameTo), 0o755); err != nil {
			return err
		}
		if _, err := os.Lstat(w.renameTo); err == nil {
			return fmt.Errorf("rename dest %s appeared before commit", w.destRel)
		}
		if err := os.Rename(w.full, w.renameTo); err != nil {
			return err
		}
		w.file.Written = true
		w.file.Removed = true
		w.file.LinesTo = 0
		w.file.SHAAfter = ""
		res.Files = append(res.Files, w.file)
		res.Files = append(res.Files, FileResult{
			Path:     w.destRel,
			Created:  true,
			Written:  true,
			LinesTo:  w.file.LinesFrom,
			SHAAfter: shaOf(w.out),
		})
		return nil
	}

	renameDest := map[string]bool{}
	for _, w := range pathOps {
		if w.destRel != "" {
			renameDest[w.destRel] = true
		}
	}
	run := func(pred func(pending) bool, fn func(pending) error) error {
		for _, w := range pathOps {
			if !pred(w) {
				continue
			}
			if err := fn(w); err != nil {
				restore()
				return fmt.Errorf("%s: %w (%s)", w.file.Path, err, writtenSoFar(res.Files))
			}
		}
		return nil
	}
	if err := run(func(w pending) bool { return w.unlink && renameDest[w.file.Path] }, unlinkOne); err != nil {
		return err
	}
	if err := run(func(w pending) bool { return w.renameTo != "" }, renameOne); err != nil {
		return err
	}
	if err := run(func(w pending) bool { return w.unlink && !renameDest[w.file.Path] }, unlinkOne); err != nil {
		return err
	}
	for _, a := range asides {
		os.Remove(a.tmp)
	}
	return nil
}
