package apply

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/lines"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/refusal"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/regular"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted"
)

// copied is one from= hunk's source as the pre-pass resolved it (ADR-167).
type copied struct {
	body        []string
	first, last string // through trim, as removed_first and removed_last are
	span        string // PATH:A-B, as the caller's path was spelled
	path        string // the cleaned source path, as the ledger is keyed
	start, end  int
}

// sourceFile is a source read once, however many hunks copy from it.
type sourceFile struct {
	full string
	text text
	sha  string
	why  string // the refusal, when the source cannot be copied at all
	kind refusal.Kind
}

// resolveSources fills each from= hunk's body from the file as it is on disk
// NOW, before any hunk is validated or anything written, so the lines are the
// original's whatever else the plan does to the source (ADR-167 Decision 1).
// The source has its own gate (Decision 3): the ledger must show the caller was
// served those lines in this version, and neither Force nor a missing
// observation nor a nil ledger waives it, because a copy forced past its
// licence is a body nobody was shown. A refused source is returned as a failed
// hunk; the caller then reports every other hunk skipped.
func resolveSources(root string, in []Input, opt Options) (map[int]copied, map[int]HunkResult) {
	var (
		ok    = map[int]copied{}
		fails = map[int]HunkResult{}
		files = map[string]*sourceFile{}
	)
	for n, i := range in {
		if i.From == "" {
			continue
		}
		fail := func(kind refusal.Kind, start, end int, format string, a ...any) {
			p := filepath.Clean(i.From)
			fails[n] = HunkResult{
				Path: filepath.Clean(i.Path), Addr: srcAddrOf(i), Op: i.Op, SrcLine: i.SrcLine,
				Status: StatusFailed, Reason: fmt.Sprintf(format, a...), Kind: kind,
				BodyFrom: p + ":" + fromAddrString(i), fromForm: true, srcPath: p, srcStart: start, srcEnd: end,
			}
		}
		// ADR-030: the engine refuses what the parser refuses, for a caller that
		// builds an Input by hand.
		switch {
		case i.Op != "replace" && i.Op != "insert-after" && i.Op != "insert-before" && i.Op != "create":
			fail("", 0, 0, "from= copies a body, and %s takes none: its body is an expected removal or a destination", i.Op)
			continue
		case len(i.Body) > 0:
			fail("", 0, 0, "from=%s:%s is the hunk's body, and %d line(s) are also given: use one", i.From, fromAddrString(i), len(i.Body))
			continue
		case !numericSpan(i):
			fail("", 0, 0, "from=%s: the span %s is not a source address (want N, N-M, A,+N, $ or N-$, counting from line 1)", i.From, fromAddrString(i))
			continue
		}
		p := filepath.Clean(i.From)
		f, seenIt := files[p]
		if !seenIt {
			f = loadSource(root, p)
			files[p] = f
		}
		if f.why != "" {
			fail(f.kind, 0, 0, "%s", f.why)
			continue
		}
		start, end, why := sourceSpan(i, len(f.text.lines))
		if why != "" {
			fail("", 0, 0, "from=%s: %s", p, why)
			continue
		}
		if reason := sourceGate(opt, p, f, start, end); reason != "" {
			fail(refusal.SourceNotRead, start, end, "%s", reason)
			continue
		}
		body := append([]string(nil), f.text.lines[start-1:end]...)
		ok[n] = copied{
			body: body, first: trim(body[0]), last: trim(body[len(body)-1]),
			span: p + ":" + addrString(start, end), path: p, start: start, end: end,
		}
	}
	return ok, fails
}

// fromAddrString renders a from= span as the caller wrote it.
func fromAddrString(i Input) string {
	s := addrString(i.FromStart, i.FromEnd)
	if i.FromRelEnd > 0 {
		s += ",+" + strconv.Itoa(i.FromRelEnd)
	}
	return s
}

// numericSpan reports whether a from= span is one of N, N-M, A,+N, $ or N-$.
func numericSpan(i Input) bool {
	switch {
	case i.FromRelEnd < 0:
		return false
	case i.FromStart == EOF:
		return i.FromEnd == EOF && i.FromRelEnd == 0
	case i.FromStart < 1:
		return false
	case i.FromRelEnd > 0:
		return i.FromEnd == i.FromStart
	}
	return i.FromEnd == EOF || i.FromEnd >= 1
}

// sourceSpan resolves a numeric span against the source's length. A range that
// runs past the last line, or ends before it starts, is refused, as a write
// refuses one (AGENTS.md §3).
func sourceSpan(i Input, total int) (start, end int, why string) {
	start, end = i.FromStart, i.FromEnd
	if start == EOF {
		start = total
	}
	switch {
	case i.FromRelEnd > 0:
		end = start + i.FromRelEnd
	case end == EOF:
		end = total
	}
	switch {
	case total == 0:
		return 0, 0, "the source has no lines to copy"
	case start > total || end > total:
		return 0, 0, fmt.Sprintf("lines %s run past the source's last line, %d", fromAddrString(i), total)
	case end < start:
		return 0, 0, fmt.Sprintf("the span %s ends before it starts", fromAddrString(i))
	}
	return start, end, ""
}

// loadSource reads a source file as apply reads a target, through the same
// bounds, and refuses what cannot be split into lines. rooted.Resolve and not
// resolve: a read of .git is served, so a copy from it may be (ADR-167
// Decision 6); only a write is refused there (ADR-143).
func loadSource(root, p string) *sourceFile {
	f := &sourceFile{}
	full, err := rooted.Resolve(root, p)
	if err != nil {
		f.why = fmt.Sprintf("from=%s: %v", p, err)
		return f
	}
	f.full = full
	info, err := os.Stat(full)
	switch {
	case os.IsNotExist(err):
		f.why = fmt.Sprintf("from=%s does not exist before the plan, so it has no original to copy", p)
		return f
	case err != nil:
		f.why = fmt.Sprintf("from=%s: %v", p, err)
		return f
	case info.IsDir():
		f.why = fmt.Sprintf("from=%s is a directory; a copy takes lines of a file", p)
		return f
	case !info.Mode().IsRegular():
		f.why = fmt.Sprintf("from=%s is %s", p, lines.NotRegular)
		return f
	case info.Size() > maxLoadBytes:
		f.why = fmt.Sprintf("from=%s is %d bytes, over the %d-byte limit mrw reads (ADR-104, ADR-107)", p, info.Size(), maxLoadBytes)
		return f
	}
	t, existed, err := readLines(full)
	var over errOverLoadLimit
	switch {
	case errors.As(err, &over):
		f.why = fmt.Sprintf("from=%s: %v", p, err)
	case errors.Is(err, regular.ErrNotRegular):
		f.why = fmt.Sprintf("from=%s is %s", p, lines.NotRegular)
	case err != nil:
		f.why = fmt.Sprintf("from=%s: %v", p, err)
	case !existed:
		f.why = fmt.Sprintf("from=%s does not exist before the plan, so it has no original to copy", p)
	case t.foreign != "":
		f.why = fmt.Sprintf("from=%s %s: mrw copies UTF-8 text line by line", p, t.foreign)
	}
	if f.why != "" {
		return f
	}
	f.text, f.sha = t, shaOf(t)
	return f
}

// sourceGate is ADR-167 Decision 3. It is not the target's `covered`, which
// passes a missing observation, Force and a nil ledger; and its refusals never
// advise --force. It returns the refusal, or "" when the caller was served
// lines start-end of the file as it is now.
func sourceGate(opt Options, p string, f *sourceFile, start, end int) string {
	if opt.Seen == nil {
		return fmt.Sprintf("%s cannot be copied from: there is no read ledger, and a copy is licensed only by lines mrw served", p)
	}
	obs, known := opt.Seen[p]
	if !known {
		obs, known = sameFileEntry(f.full, p, opt.Seen)
	}
	switch {
	case !known:
		return fmt.Sprintf("%s has not been read: lines %s cannot be copied from a file mrw has not served. Run `mrw read %s:%d-%d` first",
			p, addrString(start, end), p, start, end)
	case obs.SHA != f.sha:
		return fmt.Sprintf("%s changed since mrw last saw it (recorded %s, now %s): re-read lines %s before copying them",
			p, short(obs.SHA), short(f.sha), addrString(start, end))
	case !obs.Covers(start, end):
		return fmt.Sprintf("lines %s of %s have not been read: mrw served %s. Read them first (a line --max-cols served cut is not read: read it whole, or --quiet)",
			addrString(start, end), p, obs.Served())
	}
	return ""
}
