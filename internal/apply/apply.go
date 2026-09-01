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
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

	// Seen maps a path to the SHA-256 mrw last observed it to hold. When it is
	// non-nil, a hunk touching an EXISTING file is refused unless mrw has seen
	// that file's current contents — either because the path is absent from the
	// ledger, or because what is on disk no longer matches what was recorded.
	//
	// This is the read-before-modify guarantee. A range address like "42-58"
	// only means something in the version of the file those numbers were
	// counted in, so editing a file mrw has not seen is writing against a
	// picture that may already be wrong.
	//
	// A nil ledger disables the check entirely, which is what the engine's own
	// tests use — they construct the file and the hunks in the same breath.
	Seen map[string]string

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
		out  []string
		nl   bool
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
		full := filepath.Join(root, path)
		orig, hadNewline, existed, err := readLines(full)
		if err != nil {
			return res, fmt.Errorf("%s: %w", path, err)
		}

		fr := FileResult{Path: path, LinesFrom: len(orig)}
		if existed {
			fr.SHABefore = shaOf(orig, hadNewline)
		}

		out, ok := planFile(path, hs, orig, existed, fr.SHABefore, opt, results)
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
		nl := hadNewline || !existed
		fr.LinesTo = len(out)
		fr.SHAAfter = shaOf(out, nl)
		writes = append(writes, pending{file: fr, out: out, nl: nl})
	}

	for n, i := range in {
		r, ok := results[n]
		if !ok {
			r = HunkResult{Path: i.Path, Op: i.Op, SrcLine: i.SrcLine, Status: StatusOK}
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
			if err := writeFile(filepath.Join(root, w.file.Path), w.out, w.nl); err != nil {
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
		case recorded != shaBefore:
			// The dangerous case, and the reason the ledger is written on WRITE
			// as well as on read: mrw produced or read this file, something else
			// changed it since, and the caller's line numbers now point
			// somewhere else in a file they have not seen.
			fail(hs[0], "%s changed since mrw last saw it (recorded %s, now %s): re-read it before "+
				"editing, or pass --force to overwrite blind", path, short(recorded), short(shaBefore))
		}
		if !ok {
			return nil, false
		}
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
			if h.Lines >= 0 && h.Lines != 1 {
				fail(h, "lines=%d but %s addresses a single line", h.Lines, addrString(start, start))
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
			if !guard(start) {
				continue
			}
			resolved = append(resolved, h.resolveTo(start+1, start, "insert"))
		case "insert-before":
			if start < 1 || start > total+1 {
				fail(h, "line %d is out of range (file has %d lines)", start, total)
				continue
			}
			if !guard(start) {
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

// readLines splits a file into lines, reporting whether it ended with a newline
// and whether it existed at all. A missing file is not an error here — create
// needs to know, and so does a hunk that must fail loudly.
func readLines(path string) (lines []string, hadNewline, existed bool, err error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, false, false, nil
	}
	if err != nil {
		return nil, false, false, err
	}
	if len(b) == 0 {
		return nil, false, true, nil
	}
	s := string(b)
	hadNewline = strings.HasSuffix(s, "\n")
	if hadNewline {
		s = s[:len(s)-1]
	}
	return strings.Split(s, "\n"), hadNewline, true, nil
}

// writeFile replaces a file atomically: a crash mid-write leaves the original,
// never a half-written source file.
func writeFile(path string, lines []string, trailingNewline bool) error {
	body := strings.Join(lines, "\n")
	if trailingNewline && len(lines) > 0 {
		body += "\n"
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
	if _, err := tmp.WriteString(body); err != nil {
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

func shaOf(lines []string, trailingNewline bool) string {
	body := strings.Join(lines, "\n")
	if trailingNewline && len(lines) > 0 {
		body += "\n"
	}
	sum := sha256.Sum256([]byte(body))
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
	s = strings.TrimSpace(s)
	if len(s) > 60 {
		return s[:57] + "..."
	}
	return s
}

// short renders a sha for a message: enough to identify, short enough to read.
func short(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}
