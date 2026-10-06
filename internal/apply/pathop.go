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

// What respelling finds a rename's existing destination to be (ADR-129).
const (
	anotherEntry  = iota // a different file, or the source under another link: refuse
	theSource            // the source itself, under the same leaf: refuse
	respelled            // the source under another spelling of its leaf: rename
	otherSpelling        // the plan spells the source as its directory does not: refuse
)

// respelling says what an existing destination dst is to the source src. It
// is a respelling only when the two parents are one directory, dst is the
// source's file, the leaves differ, the directory lists the source exactly as
// the plan spells it, and no other entry there — dst's leaf, or a hard link
// under any name — is the source's file. A hard link is the case POSIX
// rename(2) "does nothing" to: under another spelling it folds to dst, and
// renaming onto it would report success and change nothing. Nothing is folded
// here: the filesystem's identity and its own listing decide (ADR-021).
func respelling(src, dst string) int {
	si, err := os.Lstat(src)
	if err != nil {
		return anotherEntry
	}
	di, err := os.Lstat(dst)
	if err != nil || !os.SameFile(si, di) {
		return anotherEntry
	}
	dir := filepath.Dir(src)
	sp, err := os.Stat(dir)
	if err != nil {
		return anotherEntry
	}
	dp, err := os.Stat(filepath.Dir(dst))
	if err != nil || !os.SameFile(sp, dp) {
		return anotherEntry
	}
	srcLeaf, dstLeaf := filepath.Base(src), filepath.Base(dst)
	if srcLeaf == dstLeaf {
		return theSource
	}
	es, err := os.ReadDir(dir)
	if err != nil {
		return anotherEntry
	}
	// Every entry that is the source's file: the source itself, and any hard
	// link to it under another name.
	listed, same := false, 0
	for _, e := range es {
		if e.Name() == dstLeaf {
			return anotherEntry
		}
		fi, err := os.Lstat(filepath.Join(dir, e.Name()))
		if err != nil {
			return anotherEntry
		}
		if os.SameFile(fi, si) {
			same++
			listed = listed || e.Name() == srcLeaf
		}
	}
	switch {
	case same > 1:
		return anotherEntry
	case !listed:
		// An undo puts the file back under the plan's spelling of the source;
		// one the directory does not hold would restore a name nobody had.
		return otherSpelling
	}
	return respelled
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
		raw := h.Body[0] // the destination exactly as written (ADR-069)
		if rooted.IsRooted(raw) {
			fail(h, "%s is absolute, and every path in a plan is relative to the root: mrw looked for %s, "+
				"which does not exist", raw, raw)
			return false
		}
		// ADR-076: `d/` names a directory; cleaned, it made a FILE named d.
		if rooted.SpelledAsDirectory(raw) {
			fail(h, "rename dest %s %v; a rename names the file it makes, such as %s", raw, rooted.ErrNotADirectory, filepath.Join(filepath.Clean(raw), filepath.Base(path)))
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
			// ADR-129: on a filesystem that folds case, A.txt finds a.txt.
			switch respelling(full, destFull) {
			case respelled:
			case theSource:
				fail(h, "rename dest %s is the source", dest)
				return false
			case otherSpelling:
				fail(h, "rename source %s is not spelled that way in its directory: name it as the directory lists it", path)
				return false
			default:
				fail(h, "rename dest %s already exists — unlink it in this plan, or pick another path", dest)
				return false
			}
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
// later failure can put it back. rec is its record's index in res.Files; key is
// its path as the plan names it.
type aside struct {
	tmp  string
	orig string
	key  string
	rec  int
}

// moved is a rename that completed, kept so a later failure can undo it. recs
// are the indices of its source and destination records in res.Files; key is
// the destination as the plan names it. The undo matches an aside to a rename
// by key, never by resolved path: the two are resolved at different moments,
// and a parent swapped between them spells one path two ways (the Codex review
// of #300).
type moved struct {
	from, to string
	key      string
	recs     [2]int
	// orig is the source's record as the edit left it, for a rename of a file
	// this plan also edited (ADR-114): the rename rewrote it in place, and an
	// undo puts it back, since the source then holds the edit again.
	orig *FileResult
}

// resolvedAt is p with its parent resolved through every link as far as it
// exists and its last component literal: the spelling the root is handed for
// an unlink or a rename, which acts on that entry itself (ADR-106).
func resolvedAt(p string) string {
	return filepath.Join(rooted.RealAsFarAsItExists(filepath.Dir(p)), filepath.Base(p))
}

func commitPathOps(tr *tree, res *Result, pathOps []pending) (string, error) {
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
			if err := commitRenameFn(tr, r.to, r.from); err != nil {
				occupied[r.key] = true
				left = append(left, fmt.Sprintf("could not move %s back to %s: %v", r.to, r.from, err))
				continue
			}
			if r.orig != nil {
				res.Files[r.recs[0]] = *r.orig
				drop[r.recs[1]] = true
				continue
			}
			drop[r.recs[0]], drop[r.recs[1]] = true, true
		}
		for i := len(asides) - 1; i >= 0; i-- {
			a := asides[i]
			if occupied[a.key] {
				left = append(left, fmt.Sprintf("%s is kept in %s, because the rename onto %s could not be undone", filepath.Base(a.orig), a.tmp, a.orig))
				res.noteIfLeft(tr, a.tmp)
				continue
			}
			if err := commitRenameFn(tr, a.tmp, a.orig); err != nil {
				left = append(left, fmt.Sprintf("could not restore %s from %s: %v", a.orig, a.tmp, err))
				res.noteIfLeft(tr, a.tmp)
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
		full := resolvedAt(w.full)
		tmp, name, err := tr.createTemp(filepath.Dir(full), ".mrw-aside-")
		if err != nil {
			return err
		}
		if err := tmp.Close(); err != nil {
			res.cleanUp(tr, name)
			return err
		}
		if err := removeFn(tr, name); err != nil {
			res.noteIfLeft(tr, name)
			return err
		}
		if err := commitRenameFn(tr, full, name); err != nil {
			return err
		}
		asides = append(asides, aside{tmp: name, orig: full, key: w.file.Path, rec: len(res.Files)})
		w.file.Written = true
		w.file.Removed = true
		w.file.LinesTo = 0
		w.file.SHAAfter = ""
		res.Files = append(res.Files, w.file)
		return nil
	}
	renameOne := func(w pending) error {
		// w.renameTo was resolved at staging (apply.go); the source is
		// resolved here, both handed to the root with no link left (ADR-106).
		from := resolvedAt(w.full)
		if err := tr.mkdirAll(filepath.Dir(w.renameTo), 0o755); err != nil {
			return err
		}
		respell := false
		if _, err := tr.lstat(w.renameTo); err == nil {
			// ADR-129: a respelling's destination is the source; anything
			// else there is a file that appeared after validation.
			if respelling(from, w.renameTo) != respelled {
				return fmt.Errorf("rename dest %s appeared before commit", w.destRel)
			}
			respell = true
		}
		if err := commitRenameFn(tr, from, w.renameTo); err != nil {
			return err
		}
		// A filesystem that reports the rename done and leaves the source's
		// spelling listed made it a no-op. Asked after the rename is recorded
		// below, so a failure here is undone with it (ADR-066).
		confirm := func() error {
			if !respell {
				return nil
			}
			es, err := tr.readDir(filepath.Dir(from))
			if err != nil {
				return fmt.Errorf("the respelling of %s cannot be confirmed: %w", filepath.Base(from), err)
			}
			for _, e := range es {
				if e.Name() == filepath.Base(from) {
					return fmt.Errorf("the filesystem kept the old spelling: %s is still listed after the rename", e.Name())
				}
			}
			return nil
		}
		dest := FileResult{
			Path:     w.destRel,
			Created:  true,
			Written:  true,
			LinesTo:  len(w.out.lines),
			SHAAfter: shaOf(w.out),
		}
		// ADR-114: an edited source already has the record its content commit
		// wrote; rewrite that one, so the receipt and the ledger name the path
		// once.
		if w.edited {
			for i := range res.Files {
				if res.Files[i].Path != w.file.Path || res.Files[i].Removed {
					continue
				}
				prev := res.Files[i]
				rec := prev
				rec.Written, rec.Removed, rec.RenamedTo, rec.LinesTo, rec.SHAAfter = true, true, w.destRel, 0, ""
				res.Files[i] = rec
				renames = append(renames, moved{from: from, to: w.renameTo, key: w.destRel, recs: [2]int{i, len(res.Files)}, orig: &prev})
				res.Files = append(res.Files, dest)
				return confirm()
			}
		}
		renames = append(renames, moved{from: from, to: w.renameTo, key: w.destRel, recs: [2]int{len(res.Files), len(res.Files) + 1}})
		w.file.Written = true
		w.file.Removed = true
		w.file.LinesTo = 0
		w.file.SHAAfter = ""
		res.Files = append(res.Files, w.file)
		res.Files = append(res.Files, dest)
		return confirm()
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
		res.cleanUp(tr, a.tmp)
	}
	return "", nil
}
