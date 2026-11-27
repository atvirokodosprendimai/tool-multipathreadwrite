package apply

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted"
)

func isPathOp(op string) bool {
	return op == "unlink" || op == "rename"
}

// nesting holds, for every path a plan writes and every rename destination,
// the cleaned plan path, so a path and a path under it are refused at
// validation with a clear message and a dry run says so (#701). It is lexical
// on purpose: where a path resolves is the filesystem's to say, through links
// that may not exist yet, so the commit asks it again after staging (see the
// recheck before the first rename in Apply). Names equal once folded nest only
// by foldClashes' rule, see sameEntry.
type nesting struct {
	root string
	prod map[string]string // foldKey of the cleaned plan path -> plan path, for a path a hunk writes
	dest map[string]string // the same for a rename destination
}

func newNesting(root string, in []Input) *nesting {
	n := &nesting{root: root, prod: map[string]string{}, dest: map[string]string{}}
	add := func(set map[string]string, p string) {
		if p == "" || p == "." {
			return
		}
		fk := foldKey(p)
		if old, ok := set[fk]; !ok || p < old {
			set[fk] = p
		}
	}
	for _, i := range in {
		switch {
		case i.Op == "unlink":
		case i.Op == "rename":
			if len(i.Body) == 1 {
				add(n.dest, filepath.Clean(i.Body[0]))
			}
		default:
			add(n.prod, filepath.Clean(i.Path))
		}
	}
	return n
}

// sameEntry says whether plan paths a and b, equal once case is folded, name
// one entry: they do when they are spelled alike, or when either is not on disk
// yet (foldClashes' rule), and otherwise when the filesystem says they are the
// same file. Two entries that exist apart, as A and a on ext4, are not.
func (n *nesting) sameEntry(a, b string) bool {
	if a == b {
		return true
	}
	ia, ea := os.Lstat(filepath.Join(n.root, a))
	ib, eb := os.Lstat(filepath.Join(n.root, b))
	if ea != nil || eb != nil {
		return true
	}
	return os.SameFile(ia, ib)
}

// above is the plan path in set that is a proper ancestor of p.
func (n *nesting) above(set map[string]string, p string) (string, bool) {
	for cur := p; ; {
		parent := filepath.Dir(cur)
		if parent == "." || parent == cur {
			return "", false
		}
		if q, ok := set[foldKey(parent)]; ok && n.sameEntry(q, parent) {
			return q, true
		}
		cur = parent
	}
}

// below is the first, by name, plan path in set that sits under p.
func (n *nesting) below(set map[string]string, p string) (string, bool) {
	var best string
	var found bool
	sep := string(filepath.Separator)
	prefix := foldKey(p) + sep
	depth := strings.Count(p, sep)
	for fk, q := range set {
		if !strings.HasPrefix(fk, prefix) || (found && q >= best) {
			continue
		}
		up := q
		for strings.Count(up, sep) > depth {
			up = filepath.Dir(up)
		}
		if n.sameEntry(p, up) {
			best, found = q, true
		}
	}
	return best, found
}

// nestsWith is above or below: a plan path in set that sits over or under p.
func (n *nesting) nestsWith(set map[string]string, p string) (string, bool) {
	if q, ok := n.above(set, p); ok {
		return q, true
	}
	return n.below(set, p)
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
		fi, err := lstatEntry(filepath.Join(dir, e.Name()))
		if errors.Is(err, fs.ErrNotExist) {
			// An entry the directory lists that is not there to open: it went
			// between the listing and now. lstatEntry has already asked again
			// the one way a name Win32 normalises away can be asked.
			continue
		}
		if err != nil {
			return anotherEntry
		}
		isSource := os.SameFile(fi, si)
		// The destination's own leaf is listed as another file: refused. As
		// the source's own entry (the plan spelled the source otherwise) it is
		// no other file, and falls to !listed below, which names the cause.
		if e.Name() == dstLeaf && !isSource {
			return anotherEntry
		}
		if isSource {
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
func planPathOp(root, path, full string, h hunk, orig []string, existed bool, shaBefore, miss string, unlinked, produced map[string]bool, destCount map[string]int, nest *nesting, covered func(hunk, int, int) (Seen, bool), fail func(hunk, string, ...any), out map[int]HunkResult) bool {
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
			fail(h, "sha=%s given but %s does not exist%s", h.SHA, asPlan(path), miss)
			return false
		case !strings.HasPrefix(shaBefore, h.SHA):
			fail(h, "file changed: sha is %s, plan expected %s", shaShown(shaBefore, h.SHA), h.SHA)
			return false
		}
	}
	if !existed {
		fail(h, "%s does not exist%s", path, miss)
		return false
	}
	total := len(orig)
	if total > 0 {
		if _, ok := covered(h, 1, total); !ok {
			return false
		}
	}
	if h.Op == "rename" {
		raw := h.Body[0] // the destination exactly as written (ADR-069)
		// ADR-076: `d/` names a directory; cleaned, it made a FILE named d.
		if rooted.SpelledAsDirectory(raw) {
			fail(h, "rename dest %s %v; a rename names the file it makes, such as %s", raw, rooted.ErrNotADirectory, asPlan(filepath.Join(filepath.Clean(raw), filepath.Base(path))))
			return false
		}
		dest := filepath.Clean(raw)
		if dest == path || dest == "." {
			fail(h, "rename dest %s is the source", asPlan(dest))
			return false
		}
		if destCount[dest] > 1 {
			fail(h, "rename dest %s is named by more than one hunk", asPlan(dest))
			return false
		}
		if produced[dest] {
			fail(h, "rename dest %s is also written by another hunk in this plan", asPlan(dest))
			return false
		}
		if other, hit := nest.nestsWith(nest.dest, dest); hit && other != dest {
			fail(h, "rename dest %s nests with rename dest %s", asPlan(dest), asPlan(other))
			return false
		}
		if p, hit := nest.nestsWith(nest.prod, dest); hit {
			fail(h, "rename dest %s nests with %s written by another hunk in this plan", asPlan(dest), asPlan(p))
			return false
		}
		destFull, err := resolve(root, dest)
		if err != nil {
			fail(h, "%s", reasonOf(err, dest))
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
				// The parent is one directory however the plan spells it, and
				// a rename changes only the leaf: a/x.txt → A/X.txt would leave
				// a/X.txt on disk under a receipt naming A/X.txt (the in-process
				// review of #346).
				if filepath.Dir(dest) != filepath.Dir(path) {
					fail(h, "rename dest %s respells the directory %s: a rename changes only the name, so spell the directory as the source does", asPlan(dest), asPlan(filepath.Dir(path)))
					return false
				}
			case theSource:
				fail(h, "rename dest %s is the source", asPlan(dest))
				return false
			case otherSpelling:
				fail(h, "rename source %s is not spelled that way in its directory: name it as the directory lists it", asPlan(path))
				return false
			default:
				fail(h, "rename dest %s already exists — unlink it in this plan, or pick another path", asPlan(dest))
				return false
			}
		} else if err != nil {
			// ADR-199: a destination below a file, whichever way the system
			// reports it, is the plan's to refuse in mrw's words.
			if why := belowAFile(root, dest, destFull); why != "" {
				fail(h, "%s", why)
				return false
			}
			if !os.IsNotExist(err) {
				fail(h, "rename dest %s cannot be used: %v", asPlan(dest), err)
				return false
			}
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
	// respell marks a case-only rename (ADR-129): a filesystem that kept the
	// original spelling made it a no-op, and the undo has nothing to move back,
	// when the entry under the original spelling is still the renamed file: ident
	// is what the renamed entry was when the rename was recorded.
	respell bool
	ident   fs.FileInfo
}

// keptSpelling says whether a case-only rename was left a no-op and still is: the
// directory lists the original spelling and what it names is the renamed file. A
// spelling alone is no proof, another file may hold it now.
func keptSpelling(tr *tree, m moved) bool {
	if !m.respell || m.ident == nil || !spelledAs(tr, m.from) {
		return false
	}
	now, err := tr.lstat(m.from)
	return err == nil && os.SameFile(now, m.ident)
}

// resolvedAt is p with its parent resolved through every link as far as it
// exists and its last component literal: the spelling the root is handed for
// an unlink or a rename, which acts on that entry itself (ADR-106).
func resolvedAt(p string) string {
	return filepath.Join(rooted.RealAsFarAsItExists(filepath.Dir(p)), filepath.Base(p))
}

// spelledAs says whether the directory lists p's leaf exactly as p spells it: on
// a filesystem that folds case, the one way to tell a name from its other spelling.
func spelledAs(tr *tree, p string) bool {
	es, err := tr.readDir(filepath.Dir(p))
	if err != nil {
		return false
	}
	for _, e := range es {
		if e.Name() == filepath.Base(p) {
			return true
		}
	}
	return false
}

// pathOpFailure is what commitPathOps returns when a step failed and its undo ran:
// the step's error and what the undo could not put back. The caller builds the
// text after the content files are put back too, so what it says of what was
// written is what is on disk (ADR-209 Decision 6).
type pathOpFailure struct {
	cause  error
	suffix string
}

func (e *pathOpFailure) Error() string { return e.cause.Error() + e.suffix }
func (e *pathOpFailure) Unwrap() error { return e.cause }

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
			// A case-only rename the filesystem kept in the original spelling: the
			// entry is already where the undo would put it, nothing to move back.
			if !keptSpelling(tr, r) {
				if err := commitRenameFn(tr.noReplace(), r.to, r.from); err != nil {
					res.noteExtraName(tr, err)
					occupied[r.key] = true
					left = append(left, fmt.Sprintf("could not move %s back to %s: %v", r.to, r.from, err))
					continue
				}
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
			if err := commitRenameFn(tr.noReplace(), a.tmp, a.orig); err != nil {
				res.noteExtraName(tr, err)
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
		if why := srcChangedSince(tr, w); why != "" {
			return fmt.Errorf("%s changed after mrw read it: %s; read it again and send the plan again", w.file.Path, why)
		}
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
		// The aside's name is fresh, so nothing there can be replaced: a plain
		// rename vacates the old name at once, where a link and a removal can leave
		// it delete-pending on a Windows that lacks POSIX delete semantics.
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
		if why := srcChangedSince(tr, w); why != "" {
			return fmt.Errorf("%s changed after mrw read it: %s; read it again and send the plan again", w.file.Path, why)
		}
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
				return fmt.Errorf("rename dest %s appeared before commit", asPlan(w.destRel))
			}
			respell = true
		}
		if err := commitRenameFn(tr.noReplace(), from, w.renameTo); err != nil {
			res.noteExtraName(tr, err)
			if errors.Is(err, fs.ErrExist) {
				return fmt.Errorf("rename dest %s appeared before commit", asPlan(w.destRel))
			}
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
		// What the renamed entry is under its new name: the undo of a case-only
		// rename tells a filesystem that kept the spelling from another file by it.
		var ident fs.FileInfo
		if respell {
			ident, _ = tr.lstat(w.renameTo)
		}
		dest := FileResult{
			Path:     w.destRel,
			Created:  true,
			Written:  true,
			LinesTo:  len(w.out.lines),
			SHAAfter: shaOf(w.out),
		}
		// ADR-214 Decision 5: an edited source's count is the one apply took from
		// the landed bytes, which a marked file's moved line 1 can make differ.
		if w.edited {
			dest.LinesTo = w.file.LinesTo
		}
		// #712: a moved link can read another file at its new place, so the
		// digest of the file it read before would be a false sha_after. Its
		// target is not read at all: not here, where the plan's later path ops
		// have not settled and a link may now lead outside the root or into
		// mrw's state (Codex on #725), and not by the drift scan or the ledger.
		// The record carries no content fields, and NotProduced: mrw wrote none
		// of those bytes, so the destination licenses nothing (AGENTS.md rule
		// 2) and a write to it needs a read first.
		if fi, lerr := tr.lstat(w.renameTo); lerr == nil && fi.Mode()&os.ModeSymlink != 0 {
			dest.LinesTo, dest.SHAAfter, dest.NotProduced = 0, "", true
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
				renames = append(renames, moved{from: from, to: w.renameTo, key: w.destRel, recs: [2]int{i, len(res.Files)}, orig: &prev, respell: respell, ident: ident})
				res.Files = append(res.Files, dest)
				return confirm()
			}
		}
		renames = append(renames, moved{from: from, to: w.renameTo, key: w.destRel, recs: [2]int{len(res.Files), len(res.Files) + 1}, respell: respell, ident: ident})
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
				return w.file.Path, &pathOpFailure{cause: err, suffix: suffix}
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

// rootedPlanPaths is ADR-215's preflight: the plan paths that are rooted,
// as failed hunks by position. It tests each hunk's RAW path and each rename's
// RAW destination with rooted.IsRooted, never filepath.IsAbs (on Windows `/x`,
// `\x` and `c:x` carry no volume and IsAbs is false for all three), and it looks
// nothing up, so a rooted spelling reaches no resolver, no stat and no file
// load. Apply runs it before the ledger-backed copies, the directory-spelling
// rule, the collision maps and a rename's source load; a plan that fails it
// reports every other hunk skipped and writes nothing.
func rootedPlanPaths(root string, in []Input) map[int]HunkResult {
	var fails map[int]HunkResult
	for n, i := range in {
		raw, dest := i.Path, false
		if !rooted.IsRooted(raw) {
			if i.Op != "rename" || len(i.Body) != 1 || !rooted.IsRooted(i.Body[0]) {
				continue
			}
			raw, dest = i.Body[0], true
		}
		if fails == nil {
			fails = map[int]HunkResult{}
		}
		fails[n] = HunkResult{
			Path: filepath.Clean(i.Path), Addr: srcAddrOf(i), Op: i.Op, SrcLine: i.SrcLine,
			Status: StatusFailed, Reason: rootedReason(root, raw, dest, i.Pointer),
		}
	}
	return fails
}

// rootedReason says why a rooted path is refused and, only where it is sure,
// what to write instead: the lexical remainder of an absolute path that
// rooted.Contains puts inside the root. It never strips a separator into a
// suggestion (`/etc/hosts` would name an unrelated etc/hosts, `/../x` would
// leave the root), and never follows a link: a symlink's entry is named, not
// its target. A spelling is offered only when it is sure to parse back as the
// same name (hintable); otherwise the refusal explains and suggests nothing.
func rootedReason(root, raw string, dest bool, pointer string) string {
	s := raw + " is absolute, and every path in a plan is relative to the root"
	if rel, ok := insideRoot(root, raw); ok && hintable(rel) {
		// A header spelling: ./ before a leading @ (which would read as a
		// pointer), then double quotes round a space or a tab. A rename
		// destination is a body line, taken literally.
		if !dest {
			if strings.HasPrefix(rel, "@") {
				rel = "./" + rel
			}
			if strings.Contains(rel, " ") {
				rel = `"` + rel + `"`
			}
		}
		s += ": for the file inside the root write " + rel
	} else {
		s += ": name the file relative to the root, without the leading separator or drive"
	}
	// The pointer expanded into the hunk's path, never into a rename's destination.
	if pointer != "" && !dest {
		s += " (the working-set entry " + pointer + " holds it: mrw iter lists it; add it again without the leading separator)"
	}
	return s
}

// hintable reports whether rel, "/"-separated, can be offered as a spelling
// the plan parser reads back as exactly rel. The header tokenizer and the body
// parser give meaning to quotes, backslashes, `=` and a leading `@@`, so only
// a conservative set is offered: letters, digits, `.`, `_`, `+`, `-`, `/`, a
// space (double-quoted in a header) and a leading `@` (spelled `./@` in a
// header; a body line that begins with one `@` is read as written).
func hintable(rel string) bool {
	for i, c := range rel {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9',
			c == '.', c == '_', c == '+', c == '-', c == '/', c == ' ':
		case c == '@' && i == 0:
		default:
			return false
		}
	}
	// A body line is trimmed before its empty test, and a header token is split
	// at spaces: a name with a leading or trailing space has no spelling.
	return rel != "" && strings.TrimSpace(rel) == rel
}

// insideRoot is the "/"-spelled remainder of raw below root, when raw names
// something strictly inside it by its spelling alone, and not a directory.
// A raw path with a `..` component names none: through a symbolic link the
// system resolves `link/../x` beside the link's target, which the lexical
// clean would not (the Codex review of #800).
func insideRoot(root, raw string) (string, bool) {
	if rooted.SpelledAsDirectory(raw) {
		return "", false
	}
	for _, part := range strings.FieldsFunc(raw, func(r rune) bool { return r == '/' || r == filepath.Separator }) {
		if part == ".." {
			return "", false
		}
	}
	c := filepath.Clean(raw)
	// The root as the CLI passes it is often ".", which no absolute path is
	// lexically inside: compare against its absolute form.
	r, err := filepath.Abs(root)
	if err != nil || !rooted.Contains(r, c) {
		return "", false
	}
	rel, err := filepath.Rel(r, c)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return Slash(rel, reasonSep), true
}
