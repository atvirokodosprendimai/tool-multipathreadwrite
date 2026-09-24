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
		// ADR-066: only "does not exist" clears a destination. Acting on
		// err == nil alone let every OTHER error through — a name too long
		// for the filesystem, a parent that is a file — and each of those
		// failed at the commit rename, after the plan's other files had
		// landed. It is a fact about the plan, so it fails the hunk here.
		if _, err := os.Lstat(destFull); err == nil && !unlinked[dest] {
			fail(h, "rename dest %s already exists — unlink it in this plan, or pick another path", dest)
			return false
		} else if err != nil && !os.IsNotExist(err) {
			fail(h, "rename dest %s cannot be used: %v", dest, err)
			return false
		}
	}
	out[h.Index] = HunkResult{
		Path: path, Addr: "-", Op: h.SrcOp, SrcLine: h.SrcLine, Status: StatusOK,
	}
	return true
}

// aside is an unlinked file moved beside its path until the plan commits, so a
// later failure can put it back. rec is its record's index in res.Files.
type aside struct {
	tmp  string
	orig string
	rec  int
}

// moved is a rename that completed, kept so a later failure can undo it. recs
// are the indices of its source and destination records in res.Files.
type moved struct {
	from, to string
	recs     [2]int
}

func commitPathOps(res *Result, pathOps []pending) (string, error) {
	var asides []aside
	var renames []moved
	// undo puts the plan's unlinks and renames back after a failure, as a
	// unit (ADR-066). Every completed rename is reversed FIRST, newest first,
	// and only then are the unlinked files restored. The old restore() did
	// only the second half, so a plan that unlinked c, renamed b onto c and
	// then failed a later rename moved the unlinked c back OVER the renamed
	// b, and b's content existed nowhere (reproduced on v1.22.3).
	//
	// No undo step overwrites a file: an aside whose path is still held by a
	// rename that could not be undone stays where it is, as a recovery file.
	// It returns what it could not undo — empty when the tree is back — and
	// drops the records of everything it did undo from res.Files.
	undo := func() string {
		drop := map[int]bool{}
		occupied := map[string]bool{}
		var left []string
		for i := len(renames) - 1; i >= 0; i-- {
			r := renames[i]
			if err := commitRenameFn(r.to, r.from); err != nil {
				occupied[r.to] = true
				left = append(left, fmt.Sprintf("could not move %s back to %s: %v", r.to, r.from, err))
				continue
			}
			drop[r.recs[0]], drop[r.recs[1]] = true, true
		}
		for i := len(asides) - 1; i >= 0; i-- {
			a := asides[i]
			if occupied[a.orig] {
				left = append(left, fmt.Sprintf("%s is kept in %s, because the rename onto %s could not be undone", filepath.Base(a.orig), a.tmp, a.orig))
				continue
			}
			if err := commitRenameFn(a.tmp, a.orig); err != nil {
				left = append(left, fmt.Sprintf("could not restore %s from %s: %v", a.orig, a.tmp, err))
				continue
			}
			drop[a.rec] = true
		}
		kept := make([]FileResult, 0, len(res.Files))
		for i, f := range res.Files {
			if !drop[i] {
				kept = append(kept, f)
			}
		}
		res.Files = kept
		return strings.Join(left, "; ")
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
		if err := commitRenameFn(w.full, name); err != nil {
			return err
		}
		asides = append(asides, aside{tmp: name, orig: w.full, rec: len(res.Files)})
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
		if err := commitRenameFn(w.full, w.renameTo); err != nil {
			return err
		}
		renames = append(renames, moved{from: w.full, to: w.renameTo, recs: [2]int{len(res.Files), len(res.Files) + 1}})
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
	// run commits one phase and, on a failure, undoes the whole of the path-op
	// commit and returns the path whose step failed, so the receipt can mark
	// that hunk failed (ADR-066).
	run := func(pred func(pending) bool, fn func(pending) error) (string, error) {
		for _, w := range pathOps {
			if !pred(w) {
				continue
			}
			if err := fn(w); err != nil {
				suffix := ""
				if left := undo(); left != "" {
					suffix = "; UNDO INCOMPLETE: " + left
				}
				return w.file.Path, fmt.Errorf("%s: %w (%s)%s", w.file.Path, err, writtenSoFar(res.Files), suffix)
			}
		}
		return "", nil
	}
	if path, err := run(func(w pending) bool { return w.unlink && renameDest[w.file.Path] }, unlinkOne); err != nil {
		return path, err
	}
	if path, err := run(func(w pending) bool { return w.renameTo != "" }, renameOne); err != nil {
		return path, err
	}
	if path, err := run(func(w pending) bool { return w.unlink && !renameDest[w.file.Path] }, unlinkOne); err != nil {
		return path, err
	}
	for _, a := range asides {
		os.Remove(a.tmp)
	}
	return "", nil
}
