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
	"sort"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// Status is a hunk's verdict.
type Status string

// Hunk verdicts. Skipped means the hunk was valid but a sibling hunk in the run
// failed, so nothing was written.
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
	Path    string `json:"path"`
	Addr    string `json:"addr"`
	Op      string `json:"op"`
	Status  Status `json:"status"`
	Reason  string `json:"reason,omitempty"`
	Removed int    `json:"removed"`
	Added   int    `json:"added"`
	SrcLine int    `json:"plan_line"`

	// RemovedFirst and RemovedLast are the first and last line a DELETE took,
	// both through trim, and empty for every other op. A delete is the only op
	// with no body, so it is the only one where the caller asserts nothing
	// about what the range held; two bounded strings — the same two whatever
	// the size of the range — are what make a wrong range visible in the
	// receipt instead of in the next build (ADR-008).
	//
	// Their PRESENCE in --json is keyed on the op, not on these being
	// non-empty: see MarshalJSON.
	RemovedFirst string `json:"removed_first"`
	RemovedLast  string `json:"removed_last"`
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
	if h.Op == "delete" {
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
}

// Result is the whole run's receipt.
type Result struct {
	Root    string       `json:"root"`
	DryRun  bool         `json:"dry_run"`
	Applied bool         `json:"applied"`
	Files   []FileResult `json:"files"`
	Hunks   []HunkResult `json:"hunks"`
	Failed  int          `json:"failed"`
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
	Path    string
	Start   int
	End     int
	Op      string
	Body    []string
	SHA     string
	Lines   int
	Anchor  string
	SrcLine int
	Index   int
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
}

// Apply validates every hunk against the working tree rooted at root and, if
// all of them pass and opt.DryRun is false, writes the results. It returns a
// receipt in every case; err is non-nil only for an I/O failure that made the
// verdict itself unknowable.
func Apply(root string, in []Input, opt Options) (Result, error) {
	dryRun := opt.DryRun
	res := Result{Root: root, DryRun: dryRun}

	byPath := map[string][]hunk{}
	var order []string
	// The hunk's identity is its POSITION in the plan, not the Index the caller
	// filled in: two inputs sharing an Index would share a slot in the verdict
	// map, and one hunk's report would silently become another's.
	for n, i := range in {
		// One file, one key. The path is cleaned so that two spellings of the
		// same file — "a.go" and "./a.go" — are one entry here and one entry
		// in the seen ledger, which is keyed the same way. Without it a file
		// read under one spelling is refused as unread under the other.
		p := filepath.Clean(i.Path)
		if _, seen := byPath[p]; !seen {
			order = append(order, p)
		}
		byPath[p] = append(byPath[p], hunk{
			Path: p, Start: i.Start, End: i.End, Op: i.Op, Body: i.Body,
			SHA: i.SHA, Lines: i.Lines, Anchor: i.Anchor,
			SrcOp: i.Op, SrcAddr: addrString(i.Start, i.End), SrcLine: i.SrcLine, Index: n,
		})
	}

	type pending struct {
		file FileResult
		out  text
		full string // the resolved absolute path this text is written to
	}
	var (
		writes []pending
		// failed holds files the plan addressed but could not validate. They
		// are reported alongside the rest so the receipt names every file the
		// run was about to touch, not only the survivors.
		failed  []FileResult
		results = map[int]HunkResult{}
	)

	for _, path := range order {
		hs := byPath[path]
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
		orig, existed, err := readLines(full)
		if err != nil {
			return res, fmt.Errorf("%s: %w", path, err)
		}

		fr := FileResult{Path: path, LinesFrom: len(orig.lines)}
		if existed {
			fr.SHABefore = shaOf(orig)
		}

		out, ok := planFile(path, hs, orig.lines, existed, fr.SHABefore, opt, results)
		if !ok {
			// The plan ADDRESSED this file even though nothing will be written
			// to it. Dropping it here is how a two-file plan reported one file
			// and left the failing one out of --json's files[] — the caller
			// then under-reads how much the run was about to touch.
			failed = append(failed, fr)
			continue
		}
		fr.Created = !existed
		// A file created by a plan always ends with a newline; an edited file
		// keeps whatever it had, so mrw never silently adds or strips one.
		final := orig
		final.final = orig.final || !existed
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
			}
		}
		// Report every addressed file, written or not, in plan order: the
		// ones that failed validation alongside the ones that would have been
		// written had their siblings passed.
		byPath := make(map[string]FileResult, len(writes)+len(failed))
		for _, w := range writes {
			byPath[w.file.Path] = w.file
		}
		for _, f := range failed {
			byPath[f.Path] = f
		}
		for _, p := range order {
			if f, ok := byPath[p]; ok {
				f.Written = false
				res.Files = append(res.Files, f)
			}
		}
		return res, nil
	}

	for _, w := range writes {
		if !dryRun {
			if err := writeFile(w.full, w.out); err != nil {
				return res, fmt.Errorf("%s: %w", w.file.Path, err)
			}
			w.file.Written = true
		}
		res.Files = append(res.Files, w.file)
	}
	res.Applied = !dryRun
	return res, nil
}

// planFile validates one file's hunks and splices its new content. It records a
// verdict for every hunk and returns ok=false if any of them failed.
func planFile(path string, hs []hunk, orig []string, existed bool, shaBefore string, opt Options, out map[int]HunkResult) ([]string, bool) {
	total := len(orig)
	ok := true
	fail := func(h hunk, format string, a ...any) {
		out[h.Index] = HunkResult{
			Path: path, Addr: h.SrcAddr, Op: h.SrcOp, SrcLine: h.SrcLine,
			Status: StatusFailed, Reason: fmt.Sprintf(format, a...),
		}
		ok = false
	}

	// READ BEFORE MODIFY. An existing file may only be edited if mrw has seen
	// what it currently holds. Checked once per file, before any hunk, because
	// it is a fact about the file rather than about a hunk — but reported
	// THROUGH the first hunk so it travels in the same receipt as every other
	// verdict, and still aborts the whole run.
	if existed && opt.Seen != nil && !opt.Force {
		switch recorded, known := opt.Seen[path]; {
		case !known:
			fail(hs[0], "%s has not been read: mrw does not know what it currently holds, and a "+
				"line address means nothing without that. Run `mrw read %s` first, or pass --force", path, path)
		case recorded.SHA != shaBefore:
			// The dangerous case, and the reason the ledger is written on WRITE
			// as well as on read: mrw produced or read this file, something else
			// changed it since, and the caller's line numbers now point
			// somewhere else in a file they have not seen.
			fail(hs[0], "%s changed since mrw last saw it (recorded %s, now %s): re-read it before "+
				"editing, or pass --force to overwrite blind", path, short(recorded.SHA), short(shaBefore))
		}
		if !ok {
			return nil, false
		}
	}

	// A hunk may only address lines the caller has actually BEEN SHOWN. The
	// ledger records what each read served, so a --stat read (which renders no
	// content) licenses nothing, and a read of lines 1-5 licenses an edit to
	// lines 1-5 and not to line 40. Whoever counted the line numbers is the
	// caller, and an address counted in lines they never saw is exactly the
	// stale picture ADR-002 exists to refuse — the same failure as an edited
	// file, one level finer.
	obs, haveObs := opt.Seen[path]
	covered := func(h hunk, from, to int) bool {
		if !haveObs || opt.Force || obs.Whole() {
			return true
		}
		if obs.Covers(from, to) {
			return true
		}
		fail(h, "%s of %s has not been read: mrw served %s. A line address means nothing in lines "+
			"you have not seen — read them, or pass --force",
			addrString(from, to), path, obs.Served())
		return false
	}

	// Resolve EOF sentinels and check each hunk in isolation.
	resolved := make([]hunk, 0, len(hs))
	for _, h := range hs {
		if h.SHA != "" {
			switch {
			case !existed:
				fail(h, "sha=%s given but %s does not exist", h.SHA, path)
				continue
			case !strings.HasPrefix(shaBefore, h.SHA):
				fail(h, "file changed: sha is %s, plan expected %s", shaBefore[:len(h.SHA)], h.SHA)
				continue
			}
		}

		if h.Op == "create" {
			if existed {
				fail(h, "create: %s already exists (%d lines) — use replace or delete", path, total)
				continue
			}
			resolved = append(resolved, h.resolveTo(1, 0, "insert"))
			continue
		}
		if !existed {
			fail(h, "%s does not exist", path)
			continue
		}

		start, end := h.Start, h.End
		if start == EOF {
			start = total
		}
		if end == EOF {
			end = total
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
			if !guard(start) || !covered(h, min(max(start, 1), total), min(max(start, 1), total)) {
				continue
			}
			resolved = append(resolved, h.resolveTo(start+1, start, "insert"))
		case "insert-before":
			if start < 1 || start > total+1 {
				fail(h, "line %d is out of range (file has %d lines)", start, total)
				continue
			}
			if !guard(start) || !covered(h, min(max(start, 1), total), min(max(start, 1), total)) {
				continue
			}
			resolved = append(resolved, h.resolveTo(start, start-1, "insert"))
		case "replace", "delete":
			if start < 1 || end > total || end < start {
				fail(h, "range %s is out of range (file has %d lines)", addrString(start, end), total)
				continue
			}
			if h.Lines >= 0 && end-start+1 != h.Lines {
				fail(h, "lines=%d but range %s covers %d line(s)", h.Lines, addrString(start, end), end-start+1)
				continue
			}
			if h.Anchor != "" && !strings.Contains(orig[start-1], h.Anchor) {
				fail(h, "anchor %q not in line %d: %s", h.Anchor, start, trim(orig[start-1]))
				continue
			}
			if !covered(h, start, end) {
				continue
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
		return nil, false
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
		switch h.Op {
		case "insert":
			res = append(res, h.Body...)
			r.Added = len(h.Body)
		case "replace":
			res = append(res, h.Body...)
			r.Removed, r.Added = h.End-h.Start+1, len(h.Body)
			cursor = h.End + 1
		case "delete":
			r.Removed = h.End - h.Start + 1
			r.RemovedFirst, r.RemovedLast = trim(orig[h.Start-1]), trim(orig[h.End-1])
			cursor = h.End + 1
		}
		out[h.Index] = r
	}
	if !ok {
		return nil, false
	}
	res = append(res, orig[cursor-1:]...)
	return res, true
}

// text is a file's contents split for editing: its lines with their terminator
// removed, the terminator itself, and whether the last line had one.
//
// The terminator is carried rather than assumed because mrw is an editor, not a
// formatter: a CRLF file must come back CRLF, and — just as important — join()
// must reproduce the ORIGINAL bytes exactly, since internal/seen hashes the raw
// file and a disagreement would make every CRLF file read as "changed behind
// mrw's back".
type text struct {
	lines []string
	eol   string
	final bool // the file ended with a terminator
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
// Three terminators are recognised, in the order that makes each unambiguous:
// a file whose every line ends "\r\n" is CRLF; a file with no "\n" at all but
// containing "\r" is the old-Mac form, whose interior would otherwise be one
// unaddressable line; anything else is LF, including a file that MIXES them —
// there a stray "\r" stays part of its line's content, which is what keeps the
// untouched lines byte-identical.
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

	s := string(b)
	t.eol = eolOf(s)
	t.final = strings.HasSuffix(s, t.eol)
	if t.final {
		s = s[:len(s)-len(t.eol)]
	}
	t.lines = strings.Split(s, t.eol)
	return t, true, nil
}

func eolOf(s string) string {
	if !strings.Contains(s, "\n") {
		if strings.Contains(s, "\r") {
			return "\r"
		}
		return "\n"
	}
	// CRLF only when EVERY newline is one: a mixed file is left alone.
	if strings.Contains(s, "\r\n") &&
		strings.Count(s, "\r\n") == strings.Count(s, "\n") {
		return "\r\n"
	}
	return "\n"
}

// writeFile replaces a file atomically: a crash mid-write leaves the original,
// never a half-written source file.
//
// The temp file is created beside the RESOLVED target. A symlink is followed
// rather than replaced — renaming over the link would leave the edit in a new
// regular file while the file the caller meant stayed untouched, and nothing in
// the receipt would say the tree's shape had changed.
func writeFile(path string, t text) error {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".mrw-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	perm := os.FileMode(0o644)
	if fi, err := os.Stat(path); err == nil {
		perm = fi.Mode().Perm()
	}
	if _, err := tmp.WriteString(t.join()); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), perm); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
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
