package read

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/lines"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// RewriteHit is one match of ast-grep --rewrite --json=stream (ADR-161). A
// pointer is nil when ast-grep left the field out, which is not the same as 0
// or "": an empty Replacement is a real one, and a missing field is refused,
// never guessed.
type RewriteHit struct {
	Text, Replacement *string
	// Start and End are range.byteOffset: UTF-8 bytes of the file, End exclusive.
	Start, End *int
}

// RewriteFile is one file as AstGrep read it, once, and the matches it kept
// for the file.
type RewriteFile struct {
	Data []byte
	Hits []RewriteHit
}

// Rewrite turns ast-grep's rewrites into a plan (ADR-161): each file's matches
// are applied to its bytes, the lines they change are grouped into whole-line
// spans, and each span is one `replace` hunk. File fills Plan, Sites, Files,
// Observed and Refused; a caller reads them after the last file.
type Rewrite struct {
	// Plan is the plan text: one hunk per span of changed lines.
	Plan strings.Builder
	// Sites and Files count the hunks and the files holding them.
	Sites, Files int
	// Observed is what the plan licenses, per file: each span and the line after
	// it, written and not shown (ADR-118), so ADR-052 never refuses the replace.
	Observed map[string]seen.Observation
	// Refused names each file the rewrite cannot be written as a plan for. When
	// any is named, the caller plans and licenses nothing at all.
	Refused []string
}

// File plans one file's matches. data is the file as AstGrep read it; every
// hunk carries its sha, so the plan applies to that version or to none. A
// refused file adds to Refused and plans nothing.
func (r *Rewrite) File(path string, data []byte, hits []RewriteHit) {
	key := filepath.Clean(path)
	sum := sha256.Sum256(data)
	sha := hex.EncodeToString(sum[:])
	none := seen.Observation{SHA: sha, Spans: [][2]int{}}
	refuse := func(format string, a ...any) {
		r.Refused = append(r.Refused, path+": "+fmt.Sprintf(format, a...))
		r.Observed[key] = none
	}
	if r.Observed == nil {
		r.Observed = map[string]seen.Observation{}
	}
	name, ok := planPath(path)
	if !ok {
		refuse("a line break in a path cannot be written in a plan header")
		return
	}
	type edit struct {
		start, end int
		repl       string
	}
	edits := make([]edit, 0, len(hits))
	for _, h := range hits {
		switch {
		case h.Replacement == nil:
			refuse("ast-grep reported a match with no replacement")
			return
		case h.Start == nil || h.End == nil:
			refuse("ast-grep reported a match with no range.byteOffset")
			return
		case *h.Start < 0 || *h.End < *h.Start || *h.End > len(data):
			refuse("a match at bytes %d-%d lies outside the file (%d bytes)", *h.Start, *h.End, len(data))
			return
		// The file is read once, here, and ast-grep read it earlier: the text it
		// matched is the check that this is the file it saw.
		case h.Text != nil && string(data[*h.Start:*h.End]) != *h.Text:
			refuse("the file is not what ast-grep read: bytes %d-%d are not its match", *h.Start, *h.End)
			return
		}
		edits = append(edits, edit{*h.Start, *h.End, *h.Replacement})
	}
	sort.SliceStable(edits, func(i, j int) bool {
		if edits[i].start != edits[j].start {
			return edits[i].start < edits[j].start
		}
		return edits[i].end < edits[j].end
	})
	for i := 1; i < len(edits); i++ {
		if edits[i].start < edits[i-1].end {
			refuse("matches at bytes %d-%d and %d-%d overlap (a nested match): ast-grep would rewrite the same text twice",
				edits[i-1].start, edits[i-1].end, edits[i].start, edits[i].end)
			return
		}
	}
	ls, eol, _ := lines.Split(string(data))
	if len(ls) == 0 || eol == "\r" {
		refuse("a file of no line, or of CR-only lines, cannot be addressed by ast-grep's rows")
		return
	}
	starts := make([]int, len(ls))
	for i, off := 0, 0; i < len(ls); i++ {
		starts[i] = off
		off += len(ls[i]) + len(eol)
	}
	textEnd := func(i int) int { return starts[i] + len(ls[i]) }
	lineAt := func(off int) int { return sort.Search(len(starts), func(i int) bool { return starts[i] > off }) - 1 }

	// A span is the lines a run of matches covers, 0-based and inclusive. A
	// span whose lines touch the next one's is merged with it: two hunks on
	// adjacent lines would each need the other's line licensed.
	type span struct {
		a, b  int
		edits []edit
	}
	var spans []span
	for _, e := range edits {
		a, last := lineAt(e.start), e.start
		if e.end > e.start {
			last = e.end - 1
		}
		b := lineAt(last)
		// A match that ends in a line break takes the next line with it, so the
		// span never ends inside a terminator the body would not carry.
		if e.end > textEnd(b) && b+1 < len(ls) {
			b++
		}
		if e.end > textEnd(b) {
			refuse("the match at bytes %d-%d ends in the last line's break, which a plan line cannot replace", e.start, e.end)
			return
		}
		if n := len(spans); n > 0 && a <= spans[n-1].b+1 {
			spans[n-1].b = max(spans[n-1].b, b)
			spans[n-1].edits = append(spans[n-1].edits, e)
			continue
		}
		spans = append(spans, span{a, b, []edit{e}})
	}

	var plan strings.Builder
	var licensed [][2]int
	for _, sp := range spans {
		var text strings.Builder
		pos := starts[sp.a]
		for _, e := range sp.edits {
			text.WriteString(string(data[pos:e.start]) + e.repl)
			pos = e.end
		}
		text.WriteString(string(data[pos:textEnd(sp.b)]))
		// Split removed the terminators, so the body carries none. In a CRLF file
		// the CR of a break the split consumed goes with its LF, and only that CR:
		// the last piece's terminator was never in the text, so a CR it ends in is
		// the replacement's own. A CR left over would be dropped by the plan parser
		// and written a byte short, so the span is refused.
		body := strings.Split(text.String(), "\n")
		for i, l := range body {
			if eol == "\r\n" && i < len(body)-1 {
				l = strings.TrimSuffix(l, "\r")
			}
			if strings.HasSuffix(l, "\r") {
				refuse("lines %d-%d: the new text ends a line in a CR byte, which a plan line cannot carry", sp.a+1, sp.b+1)
				return
			}
			body[i] = l
		}
		if strings.Join(body, "\n") == strings.Join(ls[sp.a:sp.b+1], "\n") {
			continue // the rewrite left these lines as they were
		}
		// anchor= is the span's first original line, so a wrong range fails
		// instead of overwriting; apply refuses a multi-line replace without one.
		first := ls[sp.a]
		if first == "" || strings.ContainsAny(first, "\r\n") {
			refuse("line %d cannot be named by an anchor= (it is empty, or holds a CR), so a hunk for it cannot be written", sp.a+1)
			return
		}
		anchor := `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(first) + `"`
		addr := strconv.Itoa(sp.a + 1)
		if sp.b > sp.a {
			addr += "-" + strconv.Itoa(sp.b+1)
		}
		// A counted raw body whatever the text: one that starts with @@, body=
		// or a byte order mark must not be read as a header or an option.
		fmt.Fprintf(&plan, "@@ %s %s replace sha=%s body=%d raw=true anchor=%s\n%s\n", name, addr, sha, len(body), anchor, strings.Join(body, "\n"))
		licensed = append(licensed, [2]int{sp.a + 1, min(sp.b+2, len(ls))})
	}
	if len(licensed) == 0 {
		r.Observed[key] = none
		return
	}
	r.Plan.WriteString(plan.String())
	r.Sites += len(licensed)
	r.Files++
	// Licensed and not shown (ADR-118): the plan's lines may be edited, but
	// occurrence=N counts only lines a read showed.
	r.Observed[key] = seen.Observation{SHA: sha, Spans: licensed, Written: true, Shown: [][2]int{}}
}

// boundedBuffer keeps what is written to it up to limit bytes and records that
// more came, so a stream it cannot hold is refused and never half-read.
type boundedBuffer struct {
	buf   bytes.Buffer
	limit int64
	over  bool
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if room := b.limit - int64(b.buf.Len()); int64(len(p)) > room {
		b.over = true
		p = p[:max(room, 0)]
	}
	b.buf.Write(p)
	return len(p), nil
}

// sameHits reports whether two runs of ast-grep found the same matches in a
// file. The order ast-grep reports them in is not part of the answer.
func sameHits(a, b []RewriteHit) bool {
	if len(a) != len(b) {
		return false
	}
	norm := func(hs []RewriteHit) []string {
		out := make([]string, len(hs))
		for i, h := range hs {
			f := func(p *int) string {
				if p == nil {
					return "-"
				}
				return strconv.Itoa(*p)
			}
			s := func(p *string) string {
				if p == nil {
					return "-"
				}
				return "=" + strconv.Quote(*p)
			}
			out[i] = f(h.Start) + "," + f(h.End) + "," + s(h.Text) + "," + s(h.Replacement)
		}
		sort.Strings(out)
		return out
	}
	x, y := norm(a), norm(b)
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}

// AstGrepRewrite is AstGrep for a rewrite, run twice so the plan is bound to
// one version of each file (ADR-161). The first run finds the files with hits
// and each is read; the second runs over exactly those files and each is read
// again. The two reads bracket the second run, so equal bytes mean the file
// ast-grep matched then is the one read, whatever else in it a match's text
// would not show (a code line turned into a comment keeps its bytes' offsets).
// A file whose reads differ, or whose matches differ between the runs, is a
// Problem and is left out of the specs and the files. The files returned are
// the second run's, against the bytes read after it.
//
// ast-grep drops a file it cannot read without a word (0.45.3 discards the
// error: exit 0, nothing on stderr), so a rewrite would apply the files it did
// read and call it done. The completeness of the scan is therefore checked
// here: one walk of the same paths by the same rules opens every file, and each
// one it cannot read is a Problem. That over-includes on purpose: mrw does not
// know which files ast-grep's language mapping would have searched, so an
// unreadable file of any kind fails the rewrite closed.
func AstGrepRewrite(root string, paths []string, pattern, repl string, exclude []string, opt AstGrepOptions) ([]Spec, []Problem, map[string]*RewriteFile, error) {
	first := map[string]*RewriteFile{}
	opt.Rewrite, opt.Rewrites = repl, first
	specs, problems, err := AstGrep(root, paths, pattern, exclude, opt)
	if err != nil {
		return specs, problems, first, err
	}
	_, unread, err := Walk(root, paths, WalkOptions{Pattern: regexp.MustCompile(""), Exclude: exclude, NoIgnore: opt.NoIgnore})
	if err != nil {
		return nil, nil, nil, err
	}
	named := map[string]bool{}
	for _, p := range problems {
		named[filepath.ToSlash(filepath.Clean(p.Path))] = true
	}
	for _, p := range unread {
		if !named[filepath.ToSlash(filepath.Clean(p.Path))] {
			problems = append(problems, Problem{Path: p.Path, Reason: "cannot be scanned, so ast-grep may have skipped it: " + p.Reason})
		}
	}
	if len(specs) == 0 {
		return specs, problems, first, nil
	}
	names := make([]string, len(specs))
	for i, sp := range specs {
		names[i] = sp.Path
	}
	second := map[string]*RewriteFile{}
	opt.Rewrites, opt.Skipped = second, nil
	_, again, err := AstGrep(root, names, pattern, nil, opt)
	if err != nil {
		return nil, nil, nil, err
	}
	problems = append(problems, again...)
	kept := specs[:0]
	for _, sp := range specs {
		a, b := first[sp.Path], second[sp.Path]
		switch {
		case a == nil:
			// no entry: AstGrep already named the file as a Problem
		case b == nil: // the second run found nothing there, so it read nothing
			problems = append(problems, Problem{Path: sp.Path, Reason: "ast-grep found different matches in its second run"})
		case !bytes.Equal(a.Data, b.Data):
			problems = append(problems, Problem{Path: sp.Path, Reason: "changed while ast-grep ran: the file read before and after the second run differ"})
		case !sameHits(a.Hits, b.Hits):
			problems = append(problems, Problem{Path: sp.Path, Reason: "ast-grep found different matches in its second run"})
		default:
			kept = append(kept, sp)
			continue
		}
		delete(second, sp.Path)
	}
	return kept, problems, second, nil
}
