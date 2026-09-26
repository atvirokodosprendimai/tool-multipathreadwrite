// Package apply turns a parsed plan into file writes.
//
// Two properties matter more than speed, and both come from the same failure:
// a read that returns nothing is visible, a write that changes nothing is not.
//
//  1. Every hunk is validated against the file on disk BEFORE anything is
//     written, and one failure aborts the whole run. A partially applied plan
//     is the worst outcome — worse than no change, because the caller believes
//     it succeeded.
//  2. Every hunk reports its own verdict. A four-hunk plan where one anchor
//     missed says which one, rather than printing success and moving on.
package apply

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/lines"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// Status is a hunk's verdict.
type Status string

// Hunk verdicts. Skipped means the hunk's file was not written: a sibling hunk
// failed validation or staging so nothing was written, or a later commit step
// failed and this hunk's file was never renamed into place or was put back
// (ADR-066). A hunk is ok only when its file reached disk.
const (
	StatusOK      Status = "ok"
	StatusFailed  Status = "failed"
	StatusSkipped Status = "skipped"
)

// Seen is one file's entry in the read-before-modify ledger: the sha mrw last
// observed, and the line spans it actually served. It is internal/seen's own
// type, aliased so a caller can pass a ledger straight through without a
// conversion that could only ever be the identity.
type Seen = seen.Observation

// HunkResult is the per-hunk report. Reason is empty unless Status is failed.
type HunkResult struct {
	// singleLineCode and wrapTail are ADR-056's pricing inputs, set where
	// Balance is: an ok single-line replace on a non-prose path, and whether
	// --strict-balance would have refused it. Unexported: not receipt fields.
	singleLineCode bool
	wrapTail       bool

	Path    string `json:"path"`
	Addr    string `json:"addr"`
	Op      string `json:"op"`
	Status  Status `json:"status"`
	Reason  string `json:"reason,omitempty"`
	Removed int    `json:"removed"`
	Added   int    `json:"added"`
	SrcLine int    `json:"plan_line"`

	// RemovedFirst and RemovedLast are the first and last line a DELETE took,
	// both through trim, and empty for every other op. A BODYLESS delete is
	// the one edit that consumes a range while asserting nothing about it: a
	// replace's body is those lines' replacement, and an insertion consumes no
	// range at all. Two bounded strings — the same two whatever the size of
	// the range — are what make a wrong range visible in the receipt instead
	// of in the next build. A delete that carries an expected body (T2) has
	// already said what it believed was there; these report what was (ADR-008).
	//
	// Their PRESENCE in --json is keyed on the op, not on these being
	// non-empty: see MarshalJSON.
	RemovedFirst string `json:"removed_first"`
	RemovedLast  string `json:"removed_last"`

	// Echo is the opt-in pad (ADR-052): N lines after the new body, numbered
	// as they sit in the written file, so a surviving closer is visible.
	// Empty unless Options.EchoPad > 0. A closer here does not fail the hunk.
	Echo []string `json:"echo,omitempty"`

	// Balance is ADR-054's delimiter-balance delta: for an applied hunk on a
	// non-prose path, each family among `{}` `()` `[]` whose net count in the
	// original addressed lines differs from the net in the body, rendered as
	// `{ +1 → 0}`. Empty when every family matches, on a prose path, on a
	// failed or skipped hunk, and on create. It is visibility, not a
	// checker: the hunk stays ok. Naive rune counts — braces inside string
	// literals miscount, which is why this reports rather than refuses
	// (ADR-048), and a balanced insert in the wrong place is invisible to it.
	Balance string `json:"balance,omitempty"`
}

// MarshalJSON emits removed_first/removed_last on a delete hunk and on no
// other, which is what the contract promises.
//
// `omitempty` cannot express that. It keys on the VALUE, so a delete that
// removed blank lines — a common edit — marshalled with neither field, leaving
// a consumer unable to tell it from a replace, while the human receipt line
// (keyed on the op) said `from "" to ""`. The two surfaces disagreed for the
// one delete whose bounds are least self-evident.
func (h HunkResult) MarshalJSON() ([]byte, error) {
	type wire HunkResult // no methods, so no recursion
	// AND applied. The bounds are only ever populated in the splice, which
	// runs for hunks that landed — so a failed or skipped delete would
	// otherwise marshal the pair as "" and be indistinguishable from a
	// successful delete of blank lines, which is the one case this exists to
	// make legible. Presence now means "this delete removed these lines" on
	// both surfaces (PR #11 review, N3).
	if h.Op == "delete" && h.Status == StatusOK {
		return json.Marshal(wire(h))
	}
	return json.Marshal(struct {
		wire
		RemovedFirst string `json:"removed_first,omitempty"`
		RemovedLast  string `json:"removed_last,omitempty"`
	}{wire: wire(h)})
}

// FileResult reports one file's outcome, including the SHA-256 it had before
// and after, so a caller can chain a later plan onto this one's result.
type FileResult struct {
	Path      string `json:"path"`
	Created   bool   `json:"created,omitempty"`
	Written   bool   `json:"written"`
	SHABefore string `json:"sha_before,omitempty"`
	SHAAfter  string `json:"sha_after,omitempty"`
	LinesFrom int    `json:"lines_before"`
	LinesTo   int    `json:"lines_after"`
	// Removed is true when this path was unlinked, or was the source of a rename.
	Removed bool `json:"removed,omitempty"`
	// RenamedTo is the dest path of a rename, root-relative.
	RenamedTo string `json:"renamed_to,omitempty"`
	// Target is the file a write through an in-root symlink changed, when it
	// is not Path: ADR-005 §2 follows the link, and ADR-076 names where it led.
	// Root-relative; absent otherwise.
	Target string `json:"target,omitempty"`
}

// Result is the whole run's receipt.
type Result struct {
	Root    string       `json:"root"`
	DryRun  bool         `json:"dry_run"`
	Applied bool         `json:"applied"`
	Files   []FileResult `json:"files"`
	Hunks   []HunkResult `json:"hunks"`
	Failed  int          `json:"failed"`
	// Advisories is how many ok hunks carry a Balance (ADR-055): the count
	// the summary line and every receipt consumer read, so a row that
	// reports without failing is not invisible to a caller who reads only
	// the summary. Skipped and failed hunks contribute nothing.
	Advisories int `json:"advisories"`
	// DirsCreated names the directories a create or a rename made because they
	// were not there, root-relative and parents first (ADR-076). Absent when
	// none were made, and on a dry run, which makes none.
	DirsCreated []string `json:"dirs_created,omitempty"`
	// StrictSingleLine and StrictWouldRefuse feed ADR-056's pricing of
	// --strict-balance and are NOT receipt fields: the balance rows already
	// show the hunks. StrictSingleLine counts ok single-line replaces on
	// non-prose paths (what the flag looks at); StrictWouldRefuse counts
	// those the flag would have refused. Both are zero when the flag is on,
	// because a refused hunk is not ok.
	StrictSingleLine  int `json:"-"`
	StrictWouldRefuse int `json:"-"`
}

// pending is one file the run is about to commit: a content write, an unlink,
// or a rename. Declared at package scope so commitPathOps can see it.
type pending struct {
	file     FileResult
	out      text
	full     string
	unlink   bool
	renameTo string // absolute dest; empty if not a rename
	destRel  string // root-relative dest
}

// hunk is the subset of plan.Hunk this package needs. It is declared here so
// apply does not import plan: the engine is testable with hand-built hunks, and
// the plan format can change without touching the write path.
type hunk struct {
	Path   string
	Start  int // resolved, 1-based
	End    int // resolved, 1-based inclusive; equals Start-1 for insertions
	Op     string
	Body   []string
	SHA    string
	Lines  int
	Anchor string

	// StartPat and EndPat are ADR-013's pattern address, carried through
	// unresolved. Resolution happens in the loop below, against the ORIGINAL
	// file, and the resolved span then meets the ledger check like any other.
	StartPat *regexp.Regexp
	EndPat   *regexp.Regexp
	// RelEnd carries plan.Addr.RelEnd — the `A,+N` form (ADR-026) — and is
	// applied once the start has resolved and the file's length is known.
	RelEnd int
	// CountedBody carries plan.Hunk.CountedBody — whether the caller DECLARED a
	// body= count (ADR-027). Without it here, a direct Apply caller could
	// create an empty file with no body at all, which is what the parser
	// refuses; plan.validate protects the CLI, the MCP server and the curve
	// scorer, and this protects everyone else. Same shape as the RelEnd check
	// above, and found the same way.
	CountedBody bool
	// SrcOp and SrcAddr are the op and address exactly as the caller wrote
	// them. Every verdict echoes these rather than the resolved form, so a
	// report line can be matched back to the plan line that produced it.
	SrcOp   string
	SrcAddr string
	SrcLine int
	Index   int
}

// resolveTo returns a copy of h with its address and op rewritten to the form
// the splicer works in, keeping the caller's own wording for the verdict.
func (h hunk) resolveTo(start, end int, op string) hunk {
	h.Start, h.End, h.Op = start, end, op
	return h
}

// Input is one hunk as the caller describes it, with addresses still unresolved
// (EOF sentinels intact).
type Input struct {
	Path        string
	Start       int
	End         int
	Op          string
	Body        []string
	SHA         string
	Lines       int
	Anchor      string
	StartPat    *regexp.Regexp
	EndPat      *regexp.Regexp
	RelEnd      int
	CountedBody bool
	SrcLine     int
	Index       int
}

// EOF mirrors plan.EOF; resolution happens here because only this package knows
// how long the file actually is.
const EOF = -1

// Options controls one Apply run.
type Options struct {
	// DryRun validates and computes the result without writing.
	DryRun bool

	// Seen maps a path to what mrw last OBSERVED of it: the whole-file sha, and
	// the line spans actually served to the caller. When it is non-nil, a hunk
	// touching an EXISTING file is refused unless mrw has seen that file's
	// current contents — because the path is absent from the ledger, because
	// what is on disk no longer matches what was recorded, or because the lines
	// the hunk addresses were never rendered.
	//
	// This is the read-before-modify guarantee. A range address like "42-58"
	// only means something in the version of the file those numbers were
	// counted in, and only to a caller who counted them — so a read that
	// printed nothing, or printed somewhere else, licenses nothing here.
	//
	// A nil ledger disables the check entirely, which is what the engine's own
	// tests use — they construct the file and the hunks in the same breath.
	Seen map[string]Seen

	// Force bypasses the Seen check. The escape hatch, not the habit.
	Force bool

	// EchoPad is how many lines after an applied body to attach on the
	// hunk's Echo. 0 (the default) attaches nothing. Not a checker: a
	// closer in the pad does not fail the hunk (ADR-052).
	EchoPad int

	// StrictBalance (ADR-055, opt-in) refuses a single-line replace on a
	// non-prose path whose consumed line has a non-zero delimiter net that
	// the body does not match — the wrap-tail signature. A refusal is a
	// failed hunk: siblings skip, nothing is written. Off by default; the
	// same hunk then applies with a balance row.
	StrictBalance bool
}

// Apply validates every hunk against the working tree rooted at root and, if
// all of them pass and opt.DryRun is false, writes the results. It returns a
// receipt in every case; err is non-nil only for an I/O failure that made the
// verdict itself unknowable.
//
// The advisory count is taken here, once, over whatever the run returned —
// apply has several return points (refused plan, dry run, staging failure,
// success) and a count taken at one of them is a count the others forget
// (ADR-055).
func Apply(root string, in []Input, opt Options) (Result, error) {
	res, err := apply(root, in, opt)
	for _, h := range res.Hunks {
		if h.Status != StatusOK {
			continue
		}
		if h.Balance != "" {
			res.Advisories++
		}
		if h.singleLineCode {
			res.StrictSingleLine++
		}
		if h.wrapTail {
			res.StrictWouldRefuse++
		}
	}
	return res, err
}

func apply(root string, in []Input, opt Options) (Result, error) {
	dryRun := opt.DryRun
	res := Result{Root: root, DryRun: dryRun}

	byPath := map[string][]hunk{}
	var order []string
	dirSpelled := map[string]dirSpelling{}
	// The hunk's identity is its POSITION in the plan, not the Index the caller
	// filled in: two inputs sharing an Index would share a slot in the verdict
	// map, and one hunk's report would silently become another's.
	for n, i := range in {
		// One file, one key. The path is cleaned so that two spellings of the
		// same file — "a.go" and "./a.go" — are one entry here and one entry
		// in the seen ledger, which is keyed the same way. Without it a file
		// read under one spelling is refused as unread under the other.
		p := filepath.Clean(i.Path)
		if _, ok := dirSpelled[p]; !ok && rooted.SpelledAsDirectory(i.Path) {
			dirSpelled[p] = dirSpelling{raw: i.Path, index: n}
		}
		if _, seen := byPath[p]; !seen {
			order = append(order, p)
		}
		byPath[p] = append(byPath[p], hunk{
			Path: p, Start: i.Start, End: i.End, Op: i.Op, Body: i.Body,
			SHA: i.SHA, Lines: i.Lines, Anchor: i.Anchor,
			StartPat: i.StartPat, EndPat: i.EndPat, RelEnd: i.RelEnd, CountedBody: i.CountedBody,
			SrcOp: i.Op, SrcAddr: srcAddrOf(i), SrcLine: i.SrcLine, Index: n,
		})
	}

	var (
		writes []pending
		// failed holds files the plan addressed but could not validate. They
		// are reported alongside the rest so the receipt names every file the
		// run was about to touch, not only the survivors.
		failed  []FileResult
		results = map[int]HunkResult{}
	)

	// reportAddressed fills res.Files with every file the plan addressed, in
	// plan order, none of them marked written.
	//
	// Both abort paths need it and for the same reason: ADR-001 rule 3 says
	// every file the plan addressed appears in the receipt, written or not. A
	// receipt naming only the files that validated — or only the one whose
	// staging failed — is a narrower promise than the rule makes, and leaves
	// the caller unable to tell an untouched sibling from a file the plan
	// never mentioned.
	reportAddressed := func() {
		addressed := make(map[string]FileResult, len(writes)+len(failed))
		for _, w := range writes {
			addressed[w.file.Path] = w.file
		}
		for _, f := range failed {
			addressed[f.Path] = f
		}
		for _, p := range order {
			if f, ok := addressed[p]; ok {
				f.Written = false
				res.Files = append(res.Files, f)
			}
		}
	}

	// ONE FILE, ONE SPELLING. byPath keys on the cleaned path STRING, and two
	// strings can name one inode: Same.txt and same.txt on a case-insensitive
	// filesystem, or a symlink and its target anywhere. Grouped as two files,
	// both would stage a copy of the same bytes and the second rename would
	// replace the first — two hunks reported ok, one of them silently undone
	// (ADR-021, measured 2026-09-04). So every file is stat'ed once as it is
	// resolved and compared with os.SameFile against the files already
	// grouped: the filesystem's own answer, as issue #47 chose for the ledger,
	// with nothing folded. Only an EXISTING file has an inode to compare; names
	// that do not exist yet are compared by case in foldClashes (ADR-071).
	unlinked := map[string]bool{}
	produced := map[string]bool{}
	destCount := map[string]int{}
	for _, i := range in {
		p := filepath.Clean(i.Path)
		if i.Op == "unlink" {
			unlinked[p] = true
		}
		if i.Op != "unlink" && i.Op != "rename" {
			produced[p] = true
		}
		if i.Op == "rename" && len(i.Body) == 1 {
			if d := filepath.Clean(i.Body[0]); d != "" && d != "." { // the body line as written (ADR-069)
				destCount[d]++
			}
		}
	}
	clash := foldClashes(in, existsUnder(root))
	var seenFiles []groupedFile
	for _, path := range order {
		hs := byPath[path]
		// ADR-076: a trailing separator names a directory, and the clean above
		// dropped it, so `@@ a.txt/` edited a.txt at exit 0. A plan names files,
		// so the spelling is refused whether or not the path exists.
		if spelled, ok := dirSpelled[path]; ok {
			// The refusal is carried by the hunk that spelled it (review of
			// #237): the first hunk of the file may be a sibling that did not.
			at := 0
			for k, h := range hs {
				if h.Index == spelled.index {
					at = k
				}
			}
			refuseFile(results, path, hs, at, fmt.Sprintf("%s %v; a plan edits files — name the file itself", spelled.raw, rooted.ErrNotADirectory))
			failed = append(failed, FileResult{Path: path})
			continue
		}
		full, err := resolve(root, path)
		if err != nil {
			// A path that leaves the root is refused per hunk rather than
			// aborting the run, so the receipt still reports every other file.
			for _, h := range hs {
				results[h.Index] = HunkResult{
					Path: path, Addr: h.SrcAddr, Op: h.SrcOp, SrcLine: h.SrcLine,
					Status: StatusFailed, Reason: err.Error(),
				}
			}
			failed = append(failed, FileResult{Path: path})
			continue
		}
		// ADR-071: a name that differs only by case from one an earlier hunk
		// leaves on disk, where one of the two does not exist yet. One refusal,
		// on the hunk that collides; the file's other hunks skip.
		if at := clashAt(hs, clash); at >= 0 {
			for i, h := range hs {
				r := HunkResult{Path: path, Addr: h.SrcAddr, Op: h.SrcOp, SrcLine: h.SrcLine, Status: StatusSkipped}
				if i == at {
					r.Status, r.Reason = StatusFailed, clash[h.Index]
				}
				results[h.Index] = r
			}
			failed = append(failed, FileResult{Path: path})
			continue
		}
		if info, statErr := os.Stat(full); statErr == nil {
			// ADR-073: a FIFO, a socket or a device named in a plan blocked the
			// write in os.ReadFile below. It is refused before it is opened.
			if !info.Mode().IsRegular() && !info.IsDir() {
				refuseFile(results, path, hs, 0, fmt.Sprintf("%s is %s", path, lines.NotRegular))
				failed = append(failed, FileResult{Path: path})
				continue
			}
			// ADR-076: a read-only mark is a statement by whoever set it. A
			// replace renames a new file over it and an unlink removes the
			// entry, and neither needs the file to be writable, so both went
			// through at exit 0.
			if ro := readOnlyFor(path, full, info, hs); ro != "" {
				refuseFile(results, path, hs, 0, ro)
				failed = append(failed, FileResult{Path: path})
				continue
			}
			if prior, dup := sameFileAs(info, seenFiles); dup {
				// One refusal per file, carried by its FIRST hunk (ADR-021
				// Decision 3); the file's other hunks are skipped, which is
				// how ADR-001 rule 3 has every sibling of a failure report.
				// Marking them all failed would count one refusal several
				// times in the receipt.
				for i, h := range hs {
					r := HunkResult{Path: path, Addr: h.SrcAddr, Op: h.SrcOp, SrcLine: h.SrcLine, Status: StatusSkipped}
					if i == 0 {
						r.Status = StatusFailed
						r.Reason = fmt.Sprintf("%s names the same file as %s (plan line %d); one file, one spelling per plan",
							path, prior.path, prior.line)
					}
					results[h.Index] = r
				}
				failed = append(failed, FileResult{Path: path})
				continue
			}
			seenFiles = append(seenFiles, groupedFile{path: path, info: info, line: hs[0].SrcLine})
		}
		orig, existed, err := readLines(full)
		if err != nil {
			return res, fmt.Errorf("%s: %w", path, err)
		}
		// ADR-073: a line edit to a file mrw cannot split into lines — a UTF-16
		// or UTF-32 byte-order mark, or a NUL in its first 8 KiB — rewrote it at
		// exit 0 with its encodings mixed. unlink and rename do not split it.
		// The refusal is carried by the hunk that would split the file, and the
		// receipt reports the file as it is (review of #230).
		if at := lineEditAt(hs); orig.foreign != "" && at >= 0 {
			refuseFile(results, path, hs, at, fmt.Sprintf("%s %s: mrw edits UTF-8 text line by line, and a line edit would corrupt it", path, orig.foreign))
			failed = append(failed, FileResult{Path: path, LinesFrom: len(orig.lines), SHABefore: shaOf(orig)})
			continue
		}

		fr := FileResult{Path: path, LinesFrom: len(orig.lines)}
		if existed {
			fr.SHABefore = shaOf(orig)
		}

		out, ok, kind := planFile(root, path, full, hs, orig.lines, existed, fr.SHABefore, opt, unlinked, produced, destCount, results)
		if !ok {
			// The plan ADDRESSED this file even though nothing will be written
			// to it. Dropping it here is how a two-file plan reported one file
			// and left the failing one out of --json's files[] — the caller
			// then under-reads how much the run was about to touch.
			failed = append(failed, fr)
			continue
		}
		switch kind {
		case "unlink":
			writes = append(writes, pending{file: fr, full: full, unlink: true})
			continue
		case "rename":
			destRel := filepath.Clean(hs[0].Body[0]) // not TrimSpace: "d " is a name (ADR-069)
			destFull, err := resolve(root, destRel)
			if err != nil {
				for _, h := range hs {
					results[h.Index] = HunkResult{
						Path: path, Addr: h.SrcAddr, Op: h.SrcOp, SrcLine: h.SrcLine,
						Status: StatusFailed, Reason: err.Error(),
					}
				}
				failed = append(failed, fr)
				continue
			}
			fr.RenamedTo = destRel
			writes = append(writes, pending{file: fr, out: orig, full: full, renameTo: destFull, destRel: destRel})
			continue
		}
		fr.Created = !existed
		// A file created by a plan always ends with a newline, and so does one
		// that held no lines: it had no last line whose terminator could be
		// kept (ADR-076). An edited file keeps whatever it had, so mrw never
		// silently adds or strips one.
		final := orig
		final.final = orig.final || !existed || len(orig.lines) == 0
		final = final.with(out)
		fr.LinesTo = len(out)
		fr.SHAAfter = shaOf(final)
		writes = append(writes, pending{file: fr, out: final, full: full})
	}

	for n, i := range in {
		r, ok := results[n]
		if !ok {
			r = HunkResult{Path: filepath.Clean(i.Path), Op: i.Op, SrcLine: i.SrcLine, Status: StatusOK}
			r.Addr = addrString(i.Start, i.End)
		}
		if r.Status == StatusFailed {
			res.Failed++
		}
		res.Hunks = append(res.Hunks, r)
	}
	if res.Failed > 0 {
		for i := range res.Hunks {
			if res.Hunks[i].Status == StatusOK {
				res.Hunks[i].Status = StatusSkipped
				// A pad describes a write. Skip means nothing was written,
				// so leaving Echo would report the original tail (or a
				// sibling's closer) as if the hunk had landed (ADR-052).
				// Balance is the same kind of claim about a write that did
				// not happen, and it was left in place until ADR-055 T1's
				// test asked (ADR-054 T2 said skip omits it; nothing pinned it).
				res.Hunks[i].Echo = nil
				res.Hunks[i].Balance = ""
			}
		}
		reportAddressed()
		return res, nil
	}

	if dryRun {
		for _, w := range writes {
			res.Files = append(res.Files, w.file)
		}
		res.Applied = false
		return res, nil
	}

	// TWO PHASES: stage every file beside its target, then rename them all.
	//
	// A write-then-next-file loop left the tree PARTIALLY APPLIED when a later
	// file could not be written: the files before it were already renamed into
	// place, the caller saw a bare "permission denied" and no receipt at all,
	// and — because the ledger is recorded from the returned receipt — mrw then
	// refused the next edit to a file it had changed itself, reporting it as
	// "changed since mrw last saw it". ADR-001 rule 2 says one failure aborts
	// the run; staging is what makes that hold against a FILESYSTEM failure and
	// not only a validation one.
	//
	// Staging is where the realistic failures land — an unwritable directory,
	// a read-only mount, ENOSPC — and at that point nothing has been renamed,
	// so the abort leaves the tree as it found it: temp files unlinked and
	// any directories staging created removed with them. Only the rename loop
	// can still leave it partial, and a rename that fails says which files
	// were already written rather than leaving the caller to find out.
	var content, pathOps []pending
	for _, w := range writes {
		if w.unlink || w.renameTo != "" {
			pathOps = append(pathOps, w)
		} else {
			content = append(content, w)
		}
	}
	staged := make([]staged, 0, len(content))
	// The root as staging spells every path, so a link's target and the
	// directories made are reported root-relative (ADR-076).
	absRoot, absErr := rooted.Abs(root)
	// nameDirs records the directories staging made that are still on disk:
	// every one after a commit, and after a failed commit those a committed
	// file still holds, since discard has taken back the rest (Codex review
	// of #237).
	nameDirs := func() {
		if absErr != nil {
			return
		}
		res.DirsCreated = nil
		for _, sf := range staged {
			for _, d := range sf.dirs {
				if _, err := os.Stat(d); err != nil {
					continue
				}
				if rel, err := filepath.Rel(absRoot, d); err == nil {
					res.DirsCreated = append(res.DirsCreated, rel)
				}
			}
		}
		sort.Strings(res.DirsCreated)
	}
	// ADR-004: mrw leaves nothing in the working tree. An abort must unlink
	// what it staged, or a failed plan litters .mrw-* beside every target it
	// got to — and it must take the DIRECTORIES back too. Staging a create
	// into a new path calls MkdirAll, so an abort that removed only the temp
	// files left `newdir/deep` standing: a change to the tree made by a run
	// that reported writing nothing, four lines under a comment quoting
	// ADR-004. Removing them deepest-first is what makes the abort the no-op
	// this claims it is.
	//
	// os.Remove refuses a non-empty directory, and that refusal is the guard:
	// a directory something else has put a file into is left alone rather
	// than fought over.
	discard := func(from int) {
		var dirs []string
		for _, sf := range staged[from:] {
			os.Remove(sf.tmp)
			dirs = append(dirs, sf.dirs...)
		}
		sort.Slice(dirs, func(i, j int) bool { return len(dirs[i]) > len(dirs[j]) })
		for _, d := range dirs {
			os.Remove(d)
		}
	}
	// abortStage assigns the verdicts of a staging failure: the hunks of the
	// file that could not be staged are failed with the filesystem error, and
	// every other hunk is skipped, because nothing was written.
	//
	// ADR-001 rule 3 stayed OPEN on exactly this path: a filesystem failure
	// returned here with no receipt at all, so a caller learned which error
	// occurred but not which hunks it affected or which files the plan had
	// addressed. The verdicts are assigned here rather than by the validation
	// loop above, which has already run by the time staging begins — every
	// hunk was still StatusOK, and "ok but not written" is the one lie this
	// format exists to avoid. ADR-066 shares it with the rename-directory loop.
	// commitFailed assigns the verdicts of a failure AFTER the first commit
	// rename (ADR-066). Some files may already be on disk, so the staging
	// rule — everything else skipped — would deny a write that happened:
	// a hunk whose file is recorded written stays ok, the hunk whose commit
	// failed is failed, and every other hunk is skipped. A hunk that is not
	// ok describes no write, so it carries no Echo or Balance. files[] keeps
	// the written records and lists every other addressed file unwritten.
	commitFailed := func(path string, cause, ret error) (Result, error) {
		nameDirs()
		written := map[string]bool{}
		have := map[string]bool{}
		for _, f := range res.Files {
			have[f.Path] = true
			if f.Written {
				written[f.Path] = true
			}
		}
		for i := range res.Hunks {
			h := &res.Hunks[i]
			switch {
			case h.Path == path:
				h.Status = StatusFailed
				h.Reason = cause.Error()
				res.Failed++
			case written[h.Path]:
			default:
				h.Status = StatusSkipped
			}
			if h.Status != StatusOK {
				h.Echo = nil
				h.Balance = ""
			}
		}
		for _, p := range order {
			if have[p] {
				continue
			}
			for _, w := range writes {
				if w.file.Path == p {
					f := w.file
					f.Written = false
					res.Files = append(res.Files, f)
				}
			}
		}
		return res, ret
	}
	abortStage := func(path string, err error) (Result, error) {
		for i := range res.Hunks {
			if res.Hunks[i].Path == path {
				res.Hunks[i].Status = StatusFailed
				res.Hunks[i].Reason = err.Error()
				res.Hunks[i].Echo = nil
				res.Hunks[i].Balance = ""
				res.Failed++
				continue
			}
			res.Hunks[i].Status = StatusSkipped
			res.Hunks[i].Echo = nil
			res.Hunks[i].Balance = ""
		}
		reportAddressed()
		return res, fmt.Errorf("%s: %w", path, err)
	}
	for _, w := range content {
		sf, err := stageFileFn(w.full, w.out)
		if err != nil {
			// The FAILING stage counts too. It may have created directories
			// before it failed — MkdirAll succeeds, then WriteString, Close or
			// Chmod hits ENOSPC or EIO — and dropping sf here would leave
			// exactly the directories the comment above says an abort takes
			// back. ENOSPC is one of the three failures that comment names.
			staged = append(staged, sf)
			discard(0)
			return abortStage(w.file.Path, err)
		}
		staged = append(staged, sf)
	}
	// ADR-066: a rename's destination directory is made HERE, while nothing
	// has been written, and not at commit. Made at commit, a directory that
	// could not be created (a component too long for the filesystem, a byte
	// it refuses) was found after the plan's other files were already renamed
	// into place, and every hunk still read ok. The entries are appended
	// after the content entries so staged[i] still pairs with content[i] in
	// the commit loop, and discard takes these directories back like any
	// other staged directory.
	for _, w := range pathOps {
		if w.renameTo == "" {
			continue
		}
		dir := filepath.Dir(w.renameTo)
		sf := dirsOnly(missingDirs(dir))
		err := os.MkdirAll(dir, 0o755)
		staged = append(staged, sf)
		// With its parents made, the destination is asked again: a name the
		// filesystem rejects under a parent that did not exist answered "does
		// not exist" at validation and would otherwise fail at commit, after
		// the plan's other files had landed.
		if err == nil {
			if _, lerr := os.Lstat(w.renameTo); lerr != nil && !os.IsNotExist(lerr) {
				err = lerr
			}
		}
		if err != nil {
			discard(0)
			return abortStage(w.file.Path, err)
		}
	}
	for i, w := range content {
		// ADR-071: a create whose target exists by now was made by an earlier
		// rename in this commit under a name the filesystem folds into this one
		// — ß.txt and ss.txt on APFS, the NFC and NFD spellings of one name —
		// or by another process. Renaming over it would lose that body at exit
		// 0, so the commit stops and says what already landed. foldClashes
		// refuses the case pairs before anything is written; this catches the
		// folds no name comparison can see (review of #228).
		if w.file.Created {
			if _, err := os.Stat(staged[i].target); err == nil {
				discard(i)
				err := fmt.Errorf("%s appeared before commit: another name in this plan reaches the same file, or another process created it", w.file.Path)
				return commitFailed(w.file.Path, err, fmt.Errorf("%w (%s)", err, writtenSoFar(res.Files)))
			}
		}
		if err := commitRenameFn(staged[i].tmp, staged[i].target); err != nil {
			discard(i)
			return commitFailed(w.file.Path, err, fmt.Errorf("%s: %w (%s)", w.file.Path, err, writtenSoFar(res.Files)))
		}
		w.file.Written = true
		if rel, err := filepath.Rel(absRoot, staged[i].target); absErr == nil && err == nil && !sameSpelling(rel, w.file.Path) {
			w.file.Target = rel
		}
		res.Files = append(res.Files, w.file)
	}
	if path, err := commitPathOps(&res, pathOps); err != nil {
		// The rename directories staging made for renames that never ran
		// (ADR-066 T1) are taken back; one a completed rename still uses is
		// not empty, and os.Remove leaves it.
		discard(len(content))
		return commitFailed(path, err, err)
	}
	nameDirs()
	res.Applied = true
	return res, nil
}

// writtenSoFar names the files already renamed into place when a later rename
// fails, because the tree is then partially applied and the caller cannot see
// it from the error alone. Staging makes this path rare; it does not make it
// impossible, and an unreported partial write is the thing ADR-001 rule 2
// calls worse than no change.
func writtenSoFar(files []FileResult) string {
	if len(files) == 0 {
		// The i == 0 case: the FIRST rename failed, so nothing reached the
		// tree. Correct, and the only way this branch is reached — res.Files
		// grows one entry per successful rename.
		return "nothing was written"
	}
	// Only records whose file reached disk: an addressed-but-unwritten record
	// named here would send the caller to inspect a file mrw never touched.
	var names []string
	for _, f := range files {
		if f.Written {
			names = append(names, f.Path)
		}
	}
	if len(names) == 0 {
		return "nothing was written"
	}
	return "ALREADY WRITTEN: " + strings.Join(names, ", ")
}

// planFile validates one file's hunks and splices its new content. It records a
// verdict for every hunk and returns ok=false if any of them failed.
func planFile(root, path, full string, hs []hunk, orig []string, existed bool, shaBefore string, opt Options, unlinked, produced map[string]bool, destCount map[string]int, out map[int]HunkResult) ([]string, bool, string) {
	total := len(orig)
	ok := true
	fail := func(h hunk, format string, a ...any) {
		out[h.Index] = HunkResult{
			Path: path, Addr: h.SrcAddr, Op: h.SrcOp, SrcLine: h.SrcLine,
			Status: StatusFailed, Reason: fmt.Sprintf(format, a...),
		}
		ok = false
	}

	// ONE FILE IS ONE OBSERVATION, WHATEVER THE PLAN CALLS IT (ADR-029). The
	// ledger is keyed on the path a caller typed, and a file has more than one
	// valid name: an in-root symlink, or a case-only variant where the
	// filesystem is case-insensitive. So the lookup is resolved ONCE, here,
	// alias recovery included, and every check below consumes this value.
	//
	// It used to be looked up twice — once for the file-level check, which
	// recovered an alias, and once for the per-line gate, which did not. The
	// per-line gate then read "no observation for this path" as the shape of a
	// caller who has read nothing, which the file-level check has already
	// refused, and let every line through. A write to lines the caller was
	// never served APPLIED, at exit 0.
	recorded, known := opt.Seen[path]
	if !known && opt.Seen != nil {
		// On a case-insensitive filesystem — NTFS, and APFS by default —
		// two spellings name ONE file, and the ledger keyed on the string.
		// So a file that HAD been read was refused as unread, the refusal
		// asserted something false, and the remedy it suggested added a
		// third spelling rather than resolving the mismatch (issue #47).
		//
		// Resolved with os.SameFile rather than by folding case or asking
		// what kind of filesystem this is: SameFile is the filesystem's
		// OWN answer, so on Linux — where a.txt and A.TXT really are two
		// files — it says no and nothing changes. Separator normalisation
		// already worked; this is the other axis of the same equivalence.
		//
		// Only on the miss, so the ordinary case pays nothing.
		if alias, found := sameFileEntry(full, path, opt.Seen); found {
			recorded, known = alias, true
		}
	}

	var pathLevel, lineLevel bool
	for _, h := range hs {
		if isPathOp(h.Op) {
			pathLevel = true
		} else {
			lineLevel = true
		}
	}
	if pathLevel && (lineLevel || len(hs) != 1) {
		for _, h := range hs {
			fail(h, "unlink/rename cannot mix with other hunks on %s", path)
		}
		return nil, false, ""
	}

	// READ BEFORE MODIFY. An existing file may only be edited if mrw has seen
	// what it currently holds. Checked once per file, before any hunk, because
	// it is a fact about the file rather than about a hunk — but reported
	// THROUGH the first hunk so it travels in the same receipt as every other
	// verdict, and still aborts the whole run.
	if existed && opt.Seen != nil && !opt.Force {
		switch {
		case !known:
			if pathLevel {
				fail(hs[0], "%s has not been read: mrw does not know what it currently holds. %s "+
					"takes no line address — read the path, or pass --force", path, hs[0].Op)
			} else {
				fail(hs[0], "%s has not been read: mrw does not know what it currently holds, and a "+
					"line address means nothing without that. Run `mrw read %s` first, or pass --force", path, path)
			}
		case recorded.SHA != shaBefore:
			// The dangerous case, and the reason the ledger is written on WRITE
			// as well as on read: mrw produced or read this file, something else
			// changed it since, and the caller's line numbers now point
			// somewhere else in a file they have not seen.
			fail(hs[0], "%s changed since mrw last saw it (recorded %s, now %s): re-read it before "+
				"editing, or pass --force to overwrite blind", path, short(recorded.SHA), short(shaBefore))
		}
		if !ok {
			return nil, false, ""
		}
	}

	// A hunk may only address lines the caller has actually BEEN SHOWN. The
	// ledger records what each read served, so a --stat read (which renders no
	// content) licenses nothing, and a read of lines 1-5 licenses an edit to
	// lines 1-5 and not to line 40. Whoever counted the line numbers is the
	// caller, and an address counted in lines they never saw is exactly the
	// stale picture ADR-002 exists to refuse — the same failure as an edited
	// file, one level finer.
	obs, haveObs := recorded, known
	covered := func(h hunk, from, to int) bool {
		if !haveObs || opt.Force || obs.Whole() {
			return true
		}
		if obs.Covers(from, to) {
			return true
		}
		if isPathOp(h.Op) {
			fail(h, "%s has not been fully read: mrw served %s. %s takes no line address — read "+
				"the whole path, or pass --force", path, obs.Served(), h.Op)
			return false
		}
		fail(h, "%s of %s has not been read: mrw served %s. A line address means nothing in lines "+
			"you have not seen — read them, or pass --force",
			addrString(from, to), path, obs.Served())
		return false
	}

	if pathLevel {
		if !planPathOp(root, path, full, hs[0], orig, existed, shaBefore, unlinked, produced, destCount, covered, fail, out) {
			return nil, false, ""
		}
		return nil, true, hs[0].Op
	}

	// Resolve EOF sentinels and check each hunk in isolation.
	resolved := make([]hunk, 0, len(hs))
	// ADR-071: two creates of one path were two inserts into one new file, and
	// both bodies landed under two ok verdicts. A file is created once.
	created, createdAt := false, 0
	for _, h := range hs {
		if h.SHA != "" {
			switch {
			case !existed:
				fail(h, "sha=%s given but %s does not exist", h.SHA, path)
				continue
			case !strings.HasPrefix(shaBefore, h.SHA):
				fail(h, "file changed: sha is %s, plan expected %s", shaShown(shaBefore, h.SHA), h.SHA)
				continue
			}
		}

		// THE ENGINE REFUSES WHAT THE PARSER REFUSES (ADR-030). plan.validate
		// protects the CLI, the MCP server and the curve scorer, because each
		// builds its Inputs from plan.Parse. Apply is a public entry point whose
		// doc comment says it validates every hunk, and this repository's own
		// tests build Inputs directly — so every rule of validate's that does
		// not need the parse tree is asserted again here, with validate's own
		// message, so the two sites cannot say different things about one
		// mistake. It cannot call validate: internal/apply importing
		// internal/plan inverts the dependency the split exists to keep.
		//
		// ⚠ THE LIST CAME FROM ENUMERATION, NOT MEMORY. ADR-026 closed this hole
		// for a relative end and ADR-027 for a body-less create, each assuming
		// it was the last. Driving Apply with the shapes that came to mind found
		// seven open, including a `replace` with no body that DELETED the
		// addressed lines and reported ok — and THAT pass was itself incomplete,
		// because probing shapes is not walking validate's branches. Two more
		// came out of the branch walk in review.
		//
		// ⚠ THIS BLOCK RUNS BEFORE RESOLUTION, so the pattern fields are still
		// here to be asked about — StartPat and EndPat are on Input and on the
		// hunk, and resolution happens further down. An earlier cut of this
		// comment said the opposite: that a pattern had already been resolved by
		// the time Apply could look, and that validate's `patterned` gate
		// therefore could not be mirrored. Both halves were false, and a
		// patterned create and a patterned insertion range were accepted while
		// the comment explained why they could not be checked. `patterned` is
		// not a rule at all — it is a gate choosing WHICH rule applies — and
		// both sides of it are mirrored below.
		if h.Op == "create" && len(h.Body) == 0 && !h.CountedBody {
			fail(h, "create with an empty body: say body=0 if you mean an empty file, "+
				"and check the body did not go missing if you do not")
			continue
		}
		// A pattern IS an address, so a create refuses it exactly as it refuses
		// a line number — and the engine CAN ask, because Input carries both
		// pattern fields and this block runs before resolution. An earlier cut
		// of ADR-030 named this rule "deliberately absent, unresolvable here",
		// which was false twice over: the field is right there, and a patterned
		// create was accepted. Found by the review of PR #130, on the question
		// the record said it must not get wrong.
		if h.Op == "create" && h.StartPat != nil {
			fail(h, "create takes no address, use %q", "-")
			continue
		}
		if h.Op == "create" && (h.Start != 0 || h.End != 0) {
			fail(h, "create takes no address, use %q", "-")
			continue
		}
		if h.Op == "create" && h.RelEnd > 0 {
			fail(h, "create takes no address, so it takes no relative end either: use %q", "-")
			continue
		}
		if h.Op == "create" && (h.Anchor != "" || h.Lines >= 0) {
			fail(h, "create takes no anchor= or lines= (the file must not exist yet)")
			continue
		}
		if h.Op == "insert-after" || h.Op == "insert-before" {
			// A RANGE is a range however it was written, and an insertion
			// addresses one line. The relative-end form is the same mistake
			// spelled differently, which is why it is refused beside it.
			// A RANGE is a range however it was written — two patterns, two
			// line numbers, or a relative end — and an insertion addresses one
			// line. The pattern form is checked FIRST because it is the one the
			// numeric comparison cannot see: an unresolved pattern range has
			// Start == End == 0, so it passed the check below, resolved later,
			// and then silently used the start and ignored the end the caller
			// wrote. The wording is validate's own, which differs between the
			// two forms because only one of them has a range to name yet.
			if h.EndPat != nil {
				fail(h, "%s takes a single line, not a range", h.Op)
				continue
			}
			if h.RelEnd > 0 || h.Start != h.End {
				fail(h, "%s takes a single line, not the range %s", h.Op, h.SrcAddr)
				continue
			}
			if len(h.Body) == 0 {
				fail(h, "%s with an empty body would change nothing", h.Op)
				continue
			}
		}
		// The one that matters most: a body lost in transit is indistinguishable
		// from a body never written, and this shape removed code while handing
		// back a receipt that said it succeeded.
		if h.Op == "replace" && len(h.Body) == 0 {
			fail(h, "replace with an empty body would delete %s — say delete if that is "+
				"what you mean, and check the body did not go missing if it is not", h.SrcAddr)
			continue
		}

		if h.Op == "create" {
			if created {
				fail(h, "%s is created twice in this plan (plan lines %d and %d); one create per file", path, createdAt, h.SrcLine)
				continue
			}
			created, createdAt = true, h.SrcLine
			if existed {
				fail(h, "create: %s already exists (%d lines) — use replace or delete", path, total)
				continue
			}
			resolved = append(resolved, h.resolveTo(1, 0, "insert"))
			continue
		}
		if !existed {
			// An ABSOLUTE path in a plan is joined to the root, so it names
			// something under the root that is almost never there — and "does
			// not exist" then sends the caller looking for a file they can see
			// with their own eyes. Say what actually happened instead.
			//
			// The path named here must be the JOINED one. Naming the caller's
			// own path told them their real, present file did not exist, which
			// is the same misdirection this clause was written to prevent —
			// caught in review of the read-side fix for the identical bug.
			// IsRooted, not IsAbs: on Windows `/etc/hosts` carries no volume,
			// so IsAbs is false and this clause used to be skipped there —
			// restoring the very "does not exist" misdirection it prevents.
			if rooted.IsRooted(h.Path) {
				fail(h, "%s is absolute, and every path in a plan is relative to the root: mrw looked for %s, "+
					"which does not exist", h.Path, full)
				continue
			}
			fail(h, "%s does not exist", path)
			continue
		}

		// ADR-013: a pattern address resolves HERE, against `orig` — the file
		// as it was before any hunk in this run applied, which is ADR-001's
		// rule and the reason patterns and line numbers in one plan cannot
		// disagree about what they address.
		//
		// ⚠ THE ORDER IS THE SAFETY ARGUMENT. Resolution happens before the
		// `covered()` call below, and `covered()` takes the RESOLVED span, so a
		// pattern meets the same per-line ledger check a typed number meets. A
		// pattern is not a way to edit a file the caller has not read. Moving
		// resolution after the ledger check would turn this feature into a
		// bypass — TestARegexAddressIsStillSubjectToTheLedger is what notices.
		if h.StartPat != nil {
			resolve := func(re *regexp.Regexp, which string) (int, bool) {
				at := matchLines(re, orig)
				switch len(at) {
				case 1:
					return at[0], true
				case 0:
					fail(h, "%spattern %s matched no line in %s", which, re, path)
				default:
					// Naming the lines is what lets the caller act: narrow the
					// pattern, or address by number. A refusal that only says
					// "ambiguous" leaves them guessing.
					fail(h, "%spattern %s matched %d lines in %s (%s) — narrow it, or address by line number",
						which, re, len(at), path, joinInts(at))
				}
				return 0, false
			}
			from, ok := resolve(h.StartPat, "")
			if !ok {
				continue
			}
			to := from
			if h.EndPat != nil {
				// ⚠ THE END IS A DELIMITER, NOT A SITE — the first match AT OR
				// AFTER the start, which is what ed, sed and mrw's own `read`
				// mean by /a/,/b/.
				//
				// It deliberately does NOT get the exactly-once rule. That rule
				// exists to answer WHICH SITE the caller means, and the start
				// asks and answers it; once the start is unique, "the first ^}
				// after it" is one deterministic line, not a choice among
				// candidates. Applying exactly-once here shipped in the first
				// cut and made this record's own headline example —
				// /^func X/,/^}/ — fail on any file with two functions, because
				// ^} closes both. Caught in review of PR #74.
				at := matchLines(h.EndPat, orig)
				to = 0
				for _, n := range at {
					if n >= from {
						to = n
						break
					}
				}
				if to == 0 {
					if len(at) == 0 {
						fail(h, "end pattern %s matched no line in %s", h.EndPat, path)
					} else {
						fail(h, "end pattern %s matched only above the start line %d (%s) in %s",
							h.EndPat, from, joinInts(at), path)
					}
					continue
				}
			}
			h.Start, h.End = from, to
		}

		start, end := h.Start, h.End
		if start == EOF {
			start = total
		}
		if end == EOF {
			end = total
		}

		// `A,+N` (ADR-026): the end is N lines after the resolved start.
		//
		// ⚠ IT REFUSES TO RUN PAST THE LAST LINE, where a READ clamps. That is
		// not an inconsistency between the two paths, it is each path's own
		// existing rule: `mrw read f.txt:2-99` serves what exists, while a plan
		// addressed `5-9999` is already refused as out of range. A write that
		// quietly did less than the address it was given is the failure this
		// tool exists to make visible, so the relative form is refused for the
		// same reason the explicit one is. Found by the Codex review of #125,
		// which measured the two spellings disagreeing.
		if h.RelEnd > 0 {
			// Compared before it is added, for the reason the read path gives:
			// start+RelEnd overflows at a large count and a wrapped end reads
			// as in range. Refusing needs the comparison to be right.
			if start > total || h.RelEnd > total-start {
				fail(h, "range %s is out of range (file has %d lines)", h.SrcAddr, total)
				continue
			}
			end = start + h.RelEnd
		}

		// A guard the caller wrote is checked whatever the op. An insertion
		// addresses exactly one line, so lines= must be 1 and anchor= must
		// appear in that line: an insertion at a drifted address puts the
		// right text in the wrong place exactly as a replacement does, and a
		// guard that is parsed and then discarded is worse than no guard,
		// because the caller believes the edit is pinned.
		guard := func(at int) bool {
			// lines= is "the addressed range covers exactly N lines", and an
			// insertion's address is a POSITION, not a span. A position that
			// names a real line covers one; the two boundary positions —
			// insert-after 0, and insert-before one past the last line — name
			// no line and cover zero. Accepting lines=1 at a position where
			// there is no line would let a guard assert something false.
			covers := 1
			if at < 1 || at > total {
				covers = 0
			}
			if h.Lines >= 0 && h.Lines != covers {
				fail(h, "lines=%d but %s addresses %d line(s)", h.Lines, addrString(start, start), covers)
				return false
			}
			if h.Anchor == "" {
				return true
			}
			if at < 1 || at > total {
				fail(h, "anchor %q cannot be checked: %s addresses no line of the file (it has %d)",
					h.Anchor, addrString(start, start), total)
				return false
			}
			if !strings.Contains(orig[at-1], h.Anchor) {
				fail(h, "anchor %q not in line %d: %s", h.Anchor, at, trim(orig[at-1]))
				return false
			}
			return true
		}

		switch h.Op {
		case "insert-after":
			if start < 0 || start > total {
				fail(h, "line %d is out of range (file has %d lines)", start, total)
				continue
			}
			// covered() FIRST, then guard(): the guard's anchor comparison
			// quotes the line the file holds, and only the ledger establishes
			// that the caller was served it (ADR-028). `||` short-circuits, so
			// an unserved line is refused by the ledger and the anchor is never
			// evaluated. The two insertion ops had this order the wrong way
			// round after the replace/delete fix, which is how the first cut of
			// ADR-028 could claim "every guard" while two still leaked.
			if !covered(h, min(max(start, 1), total), min(max(start, 1), total)) || !guard(start) {
				continue
			}
			resolved = append(resolved, h.resolveTo(start+1, start, "insert"))
		case "insert-before":
			if start < 1 || start > total+1 {
				fail(h, "line %d is out of range (file has %d lines)", start, total)
				continue
			}
			if !covered(h, min(max(start, 1), total), min(max(start, 1), total)) || !guard(start) {
				continue
			}
			resolved = append(resolved, h.resolveTo(start, start-1, "insert"))
		case "replace", "delete":
			// A reversed range is caught at PARSE time when both ends are
			// literal, but $ resolves here — so `$-1` on a 5-line file arrived
			// as 5-1 and was reported as "out of range", which it is not. The
			// caller needs to be told the ends are the wrong way round.
			if end < start {
				fail(h, "range %s ends before it starts (it resolved to %d-%d)", h.SrcAddr, start, end)
				continue
			}
			if start < 1 || end > total {
				fail(h, "range %s is out of range (file has %d lines)", addrString(start, end), total)
				continue
			}
			if h.Lines >= 0 && end-start+1 != h.Lines {
				fail(h, "lines=%d but range %s covers %d line(s)", h.Lines, addrString(start, end), end-start+1)
				continue
			}
			if !covered(h, start, end) {
				continue
			}
			// Checked BELOW covered() for the reason ADR-008 gives three lines
			// down about its own guard: the mismatch message quotes the line
			// the file actually holds, and only the ledger check establishes
			// that the caller was served it. Above it, a failed anchor guess
			// read back a line nobody had shown them — ADR-002 and ADR-005's
			// "mrw does not tell you what it has not shown you", false on the
			// one path that had this guard first (ADR-028).
			if h.Anchor != "" && !strings.Contains(orig[start-1], h.Anchor) {
				fail(h, "anchor %q not in line %d: %s", h.Anchor, start, trim(orig[start-1]))
				continue
			}
			// ADR-035. A replace addressing more than one line must say what it
			// is replacing. mrw models no target syntax: it puts the lines it
			// was given where it was told, so a range that is wrong by a few
			// lines writes the new body over content nobody looked at, and the
			// receipt cannot show the damage because the damage is outside the
			// lines the hunk named. Two such ranges reached a build — a short
			// address, and a stale one where the file had not changed at all
			// and only the caller's belief about which line held what was
			// wrong. `anchor=` is the only guard that catches both, because it
			// is the only one that speaks about the content AT the address:
			// `lines=` compares the address against its own arithmetic, and
			// `sha=` asks whether the whole file moved, which the stale case
			// answers "no". Both stay available and neither satisfies this.
			//
			// Scoped to replace, not delete: every measured incident is a
			// replace, a wrong delete leaves an absence rather than plausible
			// wrong content, and ADR-008 already gives delete the stronger
			// opt-in guard of an expected body. Extending it on shape alone is
			// the symmetry argument ADR-027's deferral refuses.
			//
			// Checked here, after resolution, so `3-6`, `/a/,/b/`, `A,+N` and
			// `$` are one case. In plan.validate a pattern has no span yet, so
			// the requirement would have been conditional on address form —
			// the hole ADR-026 and ADR-027 each found once.
			if h.Op == "replace" && end > start && h.Anchor == "" {
				fail(h, "replace of %s covers %d lines and carries no anchor=: say anchor=\"<text from the first line>\" so a wrong range fails instead of overwriting",
					addrString(start, end), end-start+1)
				continue
			}
			// ADR-052. A multi-line replace whose ledger never covered the
			// line after End is the wrap-tail miss: the surviving closer sits
			// there, and the receipt cannot show it. Line spans only. Force
			// does not waive a partial observation — Force is for the hunk's
			// own lines. A nil ledger still skips every ledger check, which
			// is what the engine's own tests use.
			//
			// EOF (end == last line) skips: there is no End+1. Requiring
			// Start-1 would be a different miss. Insert falls out.
			//
			// The refusal names the line number, not the closer's text:
			// quoting End+1 would read back a line nobody was served
			// (ADR-028).
			if h.Op == "replace" && end > start && end < total && haveObs && !obs.Covers(end+1, end+1) {
				fail(h, "replace of %s needs a served line after %d: read line %d (the closer sits below the body)",
					addrString(start, end), end, end+1)
				continue
			}
			// ADR-055 --strict-balance: the single-line sibling of the licence
			// above, keyed on nets rather than the ledger. A replace of ONE
			// line whose consumed line opens (or closes) more than it closes
			// (or opens), replaced by a body that does not — the wrap-tail
			// shape, three of the four field breakages. Opt-in: the general
			// "refuse on any delta" was rejected in ADR-054 (strings, generated
			// code). Prose is exempt as in ADR-054; multi-line is the licence's.
			if opt.StrictBalance && h.Op == "replace" && end == start && !IsProse(path) {
				if d := wrapTailDelta(orig[start-1], h.Body); d != "" {
					fail(h, "replace of %s under --strict-balance: %s — the replaced line's delimiters do not balance and the body does not match them (the wrap-tail shape); drop the flag for this plan if the brace is inside a string",
						addrString(start, end), d)
					continue
				}
			}
			// A delete is the only op with no body, so a body on one is not
			// content to write: it is the caller's expectation of what the
			// range holds, and it is the one guard mrw cannot compute for them
			// — everything it could derive here comes from the same bytes it
			// would check against (ADR-008). A mismatch fails the hunk, which
			// by ADR-001 abandons the whole plan.
			//
			// Checked AFTER covered() on purpose: the mismatch message quotes
			// the line the file actually holds, and only the ledger check
			// establishes that the caller was served that line (ADR-005). A
			// guard must not become the one thing that reads a file back to
			// someone who never read it.
			if h.Op == "delete" && len(h.Body) > 0 {
				if covers := end - start + 1; len(h.Body) != covers {
					fail(h, "expected removal is %d line(s) but %s covers %d",
						len(h.Body), addrString(start, end), covers)
					continue
				}
				differs := false
				for i, want := range h.Body {
					if orig[start-1+i] == want {
						continue
					}
					// Named the way an anchor failure is: the expectation
					// beside the line actually there, so one attempt is enough
					// to see which of the two is wrong.
					//
					// clip, NOT trim. The commonest mismatch a hand-written
					// body produces is whitespace — a tab against four spaces,
					// a dropped trailing space, a stray CR — and trimming both
					// sides before printing them renders the two IDENTICAL on
					// screen, which is the refusal this task's Stop Condition
					// names as worse than no guard. %q makes what is left
					// legible: a tab as \t, a CR as \r, a trailing space held
					// inside the quotes.
					fail(h, "expected removal differs at line %d: plan says %q, file has %q",
						start+i, clip(want), clip(orig[start-1+i]))
					differs = true
					break
				}
				if differs {
					continue
				}
			}
			resolved = append(resolved, h.resolveTo(start, end, h.Op))
		default:
			fail(h, "unsupported op %q", h.Op)
		}
	}
	if !ok {
		return nil, false, ""
	}

	// Order by the original line each hunk begins at. Insertions sort before
	// consuming hunks at the same position so that "insert before N" and
	// "replace N-M" compose predictably; ties break on plan order.
	sort.SliceStable(resolved, func(i, j int) bool {
		a, b := resolved[i], resolved[j]
		if a.Start != b.Start {
			return a.Start < b.Start
		}
		ai, bi := a.Op == "insert", b.Op == "insert"
		if ai != bi {
			return ai
		}
		return a.Index < b.Index
	})

	var (
		res    []string
		cursor = 1
		// padAfter is the written-file index (len(res) after this hunk's
		// body) at which a later echoPad should start. Attaching during
		// the splice used the original tail; a later hunk that rewrites
		// that tail then left the receipt describing a line the file
		// no longer holds (ADR-052).
		padAfter []struct{ index, after int }
	)
	for _, h := range resolved {
		if h.Start < cursor {
			fail(h, "overlaps an earlier hunk: %s begins before line %d, which a previous hunk already consumed",
				addrString(h.Start, h.End), cursor)
			continue
		}
		res = append(res, orig[cursor-1:h.Start-1]...)
		cursor = h.Start

		r := HunkResult{Path: path, Addr: h.SrcAddr, Op: h.SrcOp, SrcLine: h.SrcLine, Status: StatusOK}
		// The lines this hunk CONSUMES, for the balance delta: an insert
		// consumes none, so its original net is zero by construction — which
		// is exactly why a balanced insert in the wrong place is invisible to
		// arm 2 (ADR-054).
		var consumed, written []string
		switch h.Op {
		case "insert":
			res = append(res, h.Body...)
			r.Added = len(h.Body)
			written = h.Body
		case "replace":
			res = append(res, h.Body...)
			r.Removed, r.Added = h.End-h.Start+1, len(h.Body)
			consumed, written = orig[h.Start-1:h.End], h.Body
			cursor = h.End + 1
		case "delete":
			r.Removed = h.End - h.Start + 1
			r.RemovedFirst, r.RemovedLast = trim(orig[h.Start-1]), trim(orig[h.End-1])
			// A delete WRITES nothing, so its body net is zero by Decision.
			// h.Body here may be ADR-008's expected removal — the consumed
			// lines verbatim — and feeding it in made every guarded delete
			// report nothing. Found by TestRandomisedApplyBalanceFollowsTheDecision.
			consumed = orig[h.Start-1 : h.End]
			cursor = h.End + 1
		}
		// Keyed on SrcOp, not h.Op: resolve turns a create into an insert at
		// line 1 of an empty file, and a create has no replaced lines for its
		// body to be compared against — a row there would be about nothing
		// (ADR-054 §2; found in review after 3b9f772).
		if !IsProse(path) && h.SrcOp != "create" {
			r.Balance = balanceDelta(consumed, written)
			// ADR-056: price --strict-balance with the flag's own predicate.
			// With the flag ON this hunk was already refused above if it
			// matched, so an ok hunk here never counts as would-refuse.
			if h.SrcOp == "replace" && h.End == h.Start {
				r.singleLineCode = true
				r.wrapTail = wrapTailDelta(orig[h.Start-1], h.Body) != ""
			}
		}
		if opt.EchoPad > 0 && (h.Op == "replace" || h.Op == "insert") {
			padAfter = append(padAfter, struct{ index, after int }{h.Index, len(res)})
		}
		out[h.Index] = r
	}
	if !ok {
		return nil, false, ""
	}
	res = append(res, orig[cursor-1:]...)
	for _, p := range padAfter {
		r := out[p.index]
		r.Echo = echoPad(res[:p.after], res[p.after:], opt.EchoPad)
		out[p.index] = r
	}
	return res, true, ""
}

// text is a file's contents split for editing: its lines with their terminator
// removed, the terminator itself, and whether the last line had one.
//
// The terminator is carried rather than assumed because mrw is an editor, not a
// formatter: a CRLF file must come back CRLF, and — just as important — join()
// must reproduce the ORIGINAL bytes exactly, since internal/seen hashes the raw
// file and a disagreement would make every CRLF file read as "changed behind
// matchLines returns the 1-based numbers of every line the pattern matches.
//
// It returns ALL of them rather than the first, because ADR-013 makes ambiguity
// a refusal and the refusal has to say how many and where. A resolver that
// stopped at the first match could not tell a caller what it was competing with,
// and would be one edit away from silently choosing.
func matchLines(re *regexp.Regexp, lines []string) []int {
	var at []int
	for i, l := range lines {
		if re.MatchString(l) {
			at = append(at, i+1)
		}
	}
	return at
}

// echoPad is the ADR-052 opt-in pad: the first n lines of after, numbered
// as they sit in the written file (one past the body already in written).
func echoPad(written, after []string, n int) []string {
	if n <= 0 || len(after) == 0 {
		return nil
	}
	if n > len(after) {
		n = len(after)
	}
	out := make([]string, n)
	start := len(written) + 1
	for i := 0; i < n; i++ {
		out[i] = fmt.Sprintf("%5d| %s", start+i, after[i])
	}
	return out
}

// balanceDelta renders ADR-054's delimiter-balance delta between the lines a
// hunk consumed and the body it wrote: for each of `{}` `()` `[]`, the net
// (opens minus closes) in each, and a `{ +1 → 0}` entry for every family
// whose nets differ. Empty when they all match. Rune counting only — no
// lexer, so a brace inside a string literal counts (ADR-048). A family
// with matching nets says nothing; a mismatch says the file's closer count
// moved, which is the wrap-tail shape, and the hunk stays ok regardless.
func balanceDelta(consumed, body []string) string {
	type family struct{ open, close rune }
	families := []family{{'{', '}'}, {'(', ')'}, {'[', ']'}}
	net := func(lines []string, f family) int {
		n := 0
		for _, l := range lines {
			for _, c := range l {
				switch c {
				case f.open:
					n++
				case f.close:
					n--
				}
			}
		}
		return n
	}
	var parts []string
	for _, f := range families {
		before, after := net(consumed, f), net(body, f)
		if before != after {
			parts = append(parts, fmt.Sprintf("%c %+d → %+d", f.open, before, after))
		}
	}
	return strings.Join(parts, "; ")
}

// wrapTailDelta is balanceDelta narrowed to the signature --strict-balance
// refuses: it reports only families whose net in the ONE consumed line is
// non-zero and differs from the body's. A consumed line that is itself
// balanced (net 0) is never the wrap-tail shape, whatever the body does, so
// a delta there is left to the advisory row (ADR-055).
func wrapTailDelta(consumed string, body []string) string {
	type family struct{ open, close rune }
	families := []family{{'{', '}'}, {'(', ')'}, {'[', ']'}}
	net := func(lines []string, f family) int {
		n := 0
		for _, l := range lines {
			for _, c := range l {
				switch c {
				case f.open:
					n++
				case f.close:
					n--
				}
			}
		}
		return n
	}
	var parts []string
	for _, f := range families {
		before, after := net([]string{consumed}, f), net(body, f)
		if before != 0 && before != after {
			parts = append(parts, fmt.Sprintf("%c %+d → %+d", f.open, before, after))
		}
	}
	return strings.Join(parts, "; ")
}

// joinInts renders line numbers for a refusal message.
func joinInts(ns []int) string {
	parts := make([]string, len(ns))
	for i, n := range ns {
		parts[i] = strconv.Itoa(n)
	}
	return "lines " + strings.Join(parts, ", ")
}

// mrw's back".
type text struct {
	lines []string
	eol   string
	final bool // the file ended with a terminator
	// foreign says why the bytes cannot be split into lines (ADR-073), or "".
	foreign string
}

// join renders the text back to the bytes it came from.
func (t text) join() string {
	body := strings.Join(t.lines, t.eol)
	if t.final && len(t.lines) > 0 {
		body += t.eol
	}
	return body
}

// with returns a copy holding different lines and the same conventions.
func (t text) with(lines []string) text { t.lines = lines; return t }

// readLines splits a file into lines, reporting the conventions it used and
// whether it existed at all. A missing file is not an error here — create needs
// to know, and so does a hunk that must fail loudly.
//
// The terminators are lines.Split's (ADR-005 §3, moved there by ADR-065 so read,
// --grep, MCP paging and the plan compilers number a file exactly as this does).
func readLines(path string) (t text, existed bool, err error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return text{eol: "\n"}, false, nil
	}
	if err != nil {
		return text{eol: "\n"}, false, err
	}
	if len(b) == 0 {
		return text{eol: "\n"}, true, nil
	}

	t.foreign = lines.Unsplittable(b)
	t.lines, t.eol, t.final = lines.Split(string(b))
	return t, true, nil
}

// stageFileFn is the seam the staging phase is driven through. A test swaps it
// to fail on a chosen file, because the realistic trigger — an unwritable
// directory — is not one a test can rely on: `chmod 0555` is a no-op for uid 0,
// so a permission-based test silently exercises nothing when CI runs as root,
// which is the defect class this whole guard exists to catch.
var stageFileFn = stageFile

// commitRenameFn is the seam every COMMIT rename goes through: a content
// temp onto its target, an unlink's file into its aside, a rename, and each
// step of the undo that follows a failure (ADR-066). A realistic trigger — a
// directory that becomes unwritable between staging and commit — is not one a
// test can rely on, so the tests fail one chosen rename here instead.
var commitRenameFn = os.Rename

// stageFile writes t to a temp file beside the RESOLVED target, leaving the
// target untouched. It returns the temp file AND the resolved path to rename
// it onto — both, because resolving here and renaming onto the unresolved path
// would drop the edit onto the symlink itself, which is the case the split of
// this function first got wrong and TestEditingThroughASymlinkKeepsTheSymlink
// caught. Renaming is the caller's second phase; until then nothing in the
// tree has changed.
//
// The temp file is created beside the target rather than in TMPDIR so the
// rename is same-filesystem, which is what makes it atomic. A symlink is
// followed rather than replaced — renaming over the link would leave the edit
// in a new regular file while the file the caller meant stayed untouched, and
// nothing in the receipt would say the tree's shape had changed.
func stageFile(path string, t text) (staged, error) {
	// Through any link, as far as the path exists: an existing file resolves
	// whole, and a file a create is about to make resolves through the
	// directory that holds it, so a create through a linked directory is
	// staged, and reported, where it lands (ADR-076).
	path = rooted.RealAsFarAsItExists(path)
	dir := filepath.Dir(path)
	// Record the directories that are about to come into existence, before
	// creating them, because MkdirAll cannot say afterwards which ones were
	// its doing. Only these are ever removed on an abort — an ancestor that
	// was already there is not this run's to take away.
	missing := missingDirs(dir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		// It may have created part of the chain before failing, so the list
		// goes back even here.
		return staged{dirs: missing}, err
	}
	tmp, err := os.CreateTemp(dir, ".mrw-*")
	if err != nil {
		return staged{dirs: missing}, err
	}

	perm := os.FileMode(0o644)
	if fi, err := os.Stat(path); err == nil {
		perm = fi.Mode().Perm()
	}
	if _, err := tmp.WriteString(t.join()); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return staged{dirs: missing}, err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return staged{dirs: missing}, err
	}
	if err := os.Chmod(tmp.Name(), perm); err != nil {
		os.Remove(tmp.Name())
		return staged{dirs: missing}, err
	}
	// ADR-076: a staged file is a new file, so the rename that commits it
	// dropped the attributes of the one it replaces; on Windows a Hidden file
	// came out visible. Only Windows has such attributes to carry.
	if err := keepAttributes(path, tmp.Name()); err != nil {
		os.Remove(tmp.Name())
		return staged{dirs: missing}, err
	}
	return staged{tmp: tmp.Name(), target: path, dirs: missing}, nil
}

// staged is one file written but not yet renamed into place, and the
// directories that had to be created to hold it.
type staged struct {
	tmp    string   // the temp file, beside target, awaiting its rename
	target string   // the RESOLVED path to rename onto; see stageFile
	dirs   []string // directories this run created, to be taken back on abort
}

// dirsOnly is a staged entry that holds only directories: a rename's
// destination directories made during staging (ADR-066), which discard takes
// back like any other. Its tmp is empty, and os.Remove("") is a harmless no-op.
func dirsOnly(dirs []string) staged { return staged{dirs: dirs} }

// missingDirs returns dir and each of its ancestors that does not exist yet,
// nearest first. It is called BEFORE MkdirAll, which is the only moment the
// answer is knowable: afterwards every one of them exists and nothing records
// which were already there.
func missingDirs(dir string) []string {
	var missing []string
	for d := dir; ; {
		// Lstat, not Stat: a dangling symlink EXISTS. Stat reports it
		// missing, so it was listed here and discard removed it on abort —
		// deleting a link the run did not make (Codex review of #207).
		if _, err := os.Lstat(d); err == nil {
			break
		}
		missing = append(missing, d)
		parent := filepath.Dir(d)
		if parent == d {
			break
		}
		d = parent
	}
	return missing
}

func shaOf(t text) string {
	sum := sha256.Sum256([]byte(t.join()))
	return hex.EncodeToString(sum[:])
}

func addrString(start, end int) string {
	f := func(n int) string {
		if n == EOF {
			return "$"
		}
		return fmt.Sprint(n)
	}
	if start == end {
		return f(start)
	}
	return f(start) + "-" + f(end)
}

// srcAddrOf renders the address AS THE CALLER WROTE IT, which for a pattern is
// the pattern and not the line it resolves to.
//
// Every verdict echoes this so a report line can be matched back to the plan
// line that produced it. Before ADR-013 that was always a line number, and a
// patterned hunk reported `0` — the unresolved bound — which named nothing the
// caller had typed and nothing the file contained.
func srcAddrOf(i Input) string {
	var s string
	if i.StartPat == nil {
		s = addrString(i.Start, i.End)
	} else {
		s = "/" + i.StartPat.String() + "/"
		if i.EndPat != nil {
			s += ",/" + i.EndPat.String() + "/"
		}
	}
	// The receipt echoes the address the caller WROTE. Dropping the relative
	// end reported `3,+1` as `3`, which hides the span the hunk consumed.
	if i.RelEnd > 0 {
		s += ",+" + strconv.Itoa(i.RelEnd)
	}
	return s
}

func trim(s string) string {
	return clip(strings.TrimSpace(s))
}

// clip bounds a line for a message WITHOUT trimming it. It is what trim is
// built on, and the difference matters in exactly one place: a message that
// reports two lines DIFFER must not first remove the whitespace they differ in.
// A caller who typed four spaces where the file has a tab was, before this,
// told `plan says "indented", file has "indented"`.
//
// The cut is on a rune boundary. `trim`'s old byte slice split multi-byte runes
// and, since ADR-008 put trimmed lines into `--json`, encoding one produced
// U+FFFD in a machine-readable field.
func clip(s string) string {
	if len(s) <= 60 {
		return s
	}
	out := make([]rune, 0, 57)
	n := 0
	for _, r := range s {
		w := len(string(r))
		if n+w > 57 {
			break
		}
		out = append(out, r)
		n += w
	}
	return string(out) + "..."
}

// short renders a sha for a message: enough to identify, short enough to read.
func short(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}

// shaShown is the on-disk prefix to print next to a mismatched sha= guard.
// Slicing to len(want) panics when the guard is longer than SHA-256.
func shaShown(have, want string) string {
	if len(want) <= len(have) {
		return have[:len(want)]
	}
	return have
}

// resolve turns a hunk's path into the absolute file it names, and refuses one
// that leaves the root.
//
// The boundary itself lives in internal/rooted, because it was implemented here
// first and therefore held only on the write path: a read served ../outside.txt
// happily while this refused it by name.
func resolve(root, path string) (string, error) {
	full, err := rooted.Resolve(root, path)
	if err != nil {
		return "", fmt.Errorf("%w: a plan may only change files under the directory mrw was pointed at", err)
	}
	return full, nil
}

// sameFileEntry finds a ledger entry that names the SAME FILE as full under a
// different spelling, which happens on any case-insensitive filesystem.
//
// The ledger is keyed by root-relative path, and root is recovered from full by
// removing the relative part — full is built by joining the two, so this is
// exact rather than a guess.
//
// os.SameFile is deliberate: it asks the filesystem instead of encoding a
// belief about it. A caller on ext4 who genuinely has both a.txt and A.TXT gets
// false and keeps two independent entries, which is correct there and would not
// be if this folded case.
func sameFileEntry(full, path string, ledger map[string]Seen) (Seen, bool) {
	want, err := os.Stat(full)
	if err != nil {
		return Seen{}, false
	}
	root := strings.TrimSuffix(full, filepath.FromSlash(path))
	for key, obs := range ledger {
		if key == path {
			continue
		}
		got, err := os.Stat(filepath.Join(root, filepath.FromSlash(key)))
		if err != nil {
			continue
		}
		if os.SameFile(want, got) {
			return obs, true
		}
	}
	return Seen{}, false
}

// groupedFile is a plan path that resolved to an existing file, with the stat
// that identifies it and the plan line of its first hunk — what a later
// spelling of the same file is refused against.
type groupedFile struct {
	path string
	info os.FileInfo
	line int
}

// sameFileAs reports whether info names a file already in grouped, and which.
// os.SameFile is the filesystem's answer to "one file or two": true for two
// spellings on a case-insensitive filesystem and for a symlink and its target,
// false for a.txt and A.txt on ext4, where they really are two files.
func sameFileAs(info os.FileInfo, grouped []groupedFile) (groupedFile, bool) {
	for _, g := range grouped {
		if os.SameFile(info, g.info) {
			return g, true
		}
	}
	return groupedFile{}, false
}

// foldClashes returns, by input position, the refusal for each hunk whose name
// differs only by case from a name an earlier hunk leaves on disk, where at
// least one of the two does not exist yet (ADR-071).
//
// ADR-021's os.SameFile answers for files that exist; a create has no inode.
// n.txt and N.TXT created in one plan on APFS or NTFS left one file and lost
// the first body at exit 0. Validation cannot tell a directory that folds case
// from one that does not without writing a probe (ADR-004), so the names are
// compared on every platform. The names a plan leaves are every path with a
// non-path op and every rename destination; a rename's source is not one, so
// a case-only rename is not compared with itself. Two names that both exist
// are left to os.SameFile, so a.txt and A.txt on ext4 both still apply.
func foldClashes(in []Input, exists func(string) bool) map[int]string {
	type named struct {
		name string
		line int
	}
	first := map[string]named{}
	out := map[int]string{}
	for n, i := range in {
		var name string
		switch {
		case i.Op == "unlink":
			continue
		case i.Op == "rename":
			if len(i.Body) != 1 {
				continue
			}
			name = filepath.Clean(i.Body[0])
		default:
			name = filepath.Clean(i.Path)
		}
		if name == "." {
			continue
		}
		key := foldKey(name)
		prior, seen := first[key]
		if !seen {
			first[key] = named{name, i.SrcLine}
			continue
		}
		if prior.name == name || (exists(prior.name) && exists(name)) {
			continue
		}
		out[n] = fmt.Sprintf("%s may name the same file as %s (plan line %d) on a case-insensitive filesystem; one file, one spelling per plan",
			name, prior.name, prior.line)
	}
	return out
}

// existsUnder reports whether a plan name is on disk under root. A name that
// leaves the root counts as present: it is refused elsewhere, and there is
// nothing to compare it with here.
func existsUnder(root string) func(string) bool {
	return func(name string) bool {
		full, err := resolve(root, name)
		if err != nil {
			return true
		}
		_, err = os.Lstat(full)
		return err == nil
	}
}

// clashAt is the index in hs of the first hunk foldClashes refused, or -1.
func clashAt(hs []hunk, clash map[int]string) int {
	for i, h := range hs {
		if _, ok := clash[h.Index]; ok {
			return i
		}
	}
	return -1
}

// foldKey is name with every rune replaced by the smallest rune of its Unicode
// simple case-folding orbit — the equality strings.EqualFold uses — so s and ſ,
// σ and ς, K and the Kelvin sign are one key. strings.ToLower is not the
// filesystem's fold: s.txt and ſ.txt passed it and APFS kept one of them
// (review of #228). Full folding (ß and ss) and normalization (NFC and NFD)
// need tables the standard library does not carry; the commit catches those.
func foldKey(name string) string {
	var b strings.Builder
	for _, r := range name {
		least := r
		for f := unicode.SimpleFold(r); f != r; f = unicode.SimpleFold(f) {
			if f < least {
				least = f
			}
		}
		b.WriteRune(least)
	}
	return b.String()
}

// refuseFile fails hs[at] with reason and skips the file's other hunks: one
// refusal per file, the shape ADR-021 Decision 3 set, reused by ADR-073.
func refuseFile(results map[int]HunkResult, path string, hs []hunk, at int, reason string) {
	for i, h := range hs {
		r := HunkResult{Path: path, Addr: h.SrcAddr, Op: h.SrcOp, SrcLine: h.SrcLine, Status: StatusSkipped}
		if i == at {
			r.Status, r.Reason = StatusFailed, reason
		}
		results[h.Index] = r
	}
}

// lineEditAt is the first of a file's hunks that edits it by line, or -1
// (ADR-073). An unlink or a rename moves the file whole, and a create over an
// existing file is refused later with its own reason, which this must not hide
// (review of #230).
func lineEditAt(hs []hunk) int {
	for i, h := range hs {
		if h.Op != "unlink" && h.Op != "rename" && h.Op != "create" {
			return i
		}
	}
	return -1
}

// readOnlyFor is the refusal for a file whose owner cannot write it, or "". A
// line edit judges the file it reaches, through a link; an unlink or a rename
// moves the entry itself, so it is judged by Lstat, and a link to a read-only
// file may still be removed (ADR-076). The fix it names is this platform's.
func readOnlyFor(path, full string, info os.FileInfo, hs []hunk) string {
	// A create over an existing file is refused for that, with its own
	// reason; the read-only mark must not hide it (review of #237).
	if allCreates(hs) {
		return ""
	}
	mode := info.Mode()
	if lineEditAt(hs) < 0 {
		li, err := os.Lstat(full)
		if err != nil {
			return ""
		}
		mode = li.Mode()
	}
	if !mode.IsRegular() || mode.Perm()&0o200 != 0 {
		return ""
	}
	fix := "chmod u+w " + path
	if runtime.GOOS == "windows" {
		fix = "attrib -r " + path
	}
	return fmt.Sprintf("%s is read-only (%v); mrw does not override a read-only file — clear the mark first: %s", path, mode.Perm(), fix)
}

// sameSpelling reports whether a staged file's path is the plan's own spelling.
// On Windows EvalSymlinks canonicalises a name's case, so readme.md staged as
// README.md is the same file reached by its own name, not through a link — a
// target named there was a false one (Codex review of #237). A link and its
// target differing only by case cannot share a directory on a filesystem that
// folds case.
func sameSpelling(staged, planned string) bool {
	return staged == planned || (runtime.GOOS == "windows" && strings.EqualFold(staged, planned))
}

// allCreates reports whether every one of a file's hunks is a create.
func allCreates(hs []hunk) bool {
	for _, h := range hs {
		if h.Op != "create" {
			return false
		}
	}
	return true
}

// dirSpelling is a plan path spelled as a directory, and the hunk that spelled
// it.
type dirSpelling struct {
	raw   string
	index int
}
