// Package plan parses mrw edit plans: a line-oriented format for describing
// several range edits across several files in one document.
//
// The format is deliberately not JSON. A plan is written by an LLM, and output
// tokens are the scarce currency — JSON would escape every newline and quote in
// every code body, inflating the one part of the document that is already large.
// The `@@` header line carries all the structure; everything after it is the
// body, verbatim.
//
//	# comment
//	@@ <path> <addr> <op> [key=value ...]
//	<body line>
//	<body line>
//	@@ ...
//
// Addresses are 1-based and inclusive, and every address in a plan resolves
// against the ORIGINAL file — never against the partially edited one. That is
// what lets a caller read a file once, note several line ranges, and write them
// all in one plan without recomputing offsets after each hunk.
package plan

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// EOF is the sentinel address component meaning "the last line of the file".
// It is resolved against the file's real length at apply time, so a plan
// written against a file of unknown length is still exact.
const EOF = -1

// Op is the kind of change a hunk makes.
type Op string

// The operations a hunk may perform. Anything else in the op column is a parse
// error: silently ignoring an unknown op would apply a plan the caller did not
// write.
const (
	OpReplace      Op = "replace"
	OpInsertAfter  Op = "insert-after"
	OpInsertBefore Op = "insert-before"
	OpDelete       Op = "delete"
	OpCreate       Op = "create"
)

// Addr is an inclusive 1-based line range, or a pair of patterns that resolve
// to one. Start may be 0 for the position before the first line, and either
// bound may be EOF.
//
// ⚠ AN ADDRESS IS LINES OR PATTERNS, NEVER BOTH. When StartPat is non-nil the
// line bounds are meaningless and whoever reads the file fills them in; when it
// is nil the patterns are absent. A half-set pair is the shape that invites a
// resolver to read the wrong field, so ParseAddr never produces one and
// TestAPatternAddressParses asserts it.
//
// Patterns are compiled HERE, at parse time, so a plan carrying an
// uncompilable one is refused as a malformed document before anything touches
// the tree — a cheap refusal, and a parse error rather than a hunk failure.
// ADR-013 records the decision; resolving a pattern to a line belongs to
// internal/apply, which is the only package that knows how long a file is.
type Addr struct {
	Start int
	End   int

	// StartPat and EndPat are the pattern form. EndPat is nil for the
	// single-pattern address `/re/`; both are nil for a line address.
	StartPat *regexp.Regexp
	EndPat   *regexp.Regexp
}

// String renders an address in the same syntax the parser accepts, so a
// diagnostic can be pasted back into a plan.
func (a Addr) String() string {
	f := func(n int) string {
		if n == EOF {
			return "$"
		}
		return strconv.Itoa(n)
	}
	if a.Start == a.End {
		return f(a.Start)
	}
	return f(a.Start) + "-" + f(a.End)
}

// Hunk is one change to one file.
type Hunk struct {
	Path string
	Addr Addr
	Op   Op
	Body []string

	// SHA, if set, is a prefix of the file's SHA-256 that must match before any
	// hunk in that file applies. It is the whole-file precondition.
	SHA string
	// Lines, if not -1, is how many lines the addressed range must contain.
	// A cheap guard: it costs one token to write and catches the drifted line
	// numbers that a body-less range edit would otherwise silently mangle.
	Lines int
	// Anchor, if set, must appear in the first line of the addressed range.
	// This is the property Edit has and Write lacks — a wrong address fails
	// loudly instead of overwriting the wrong lines.
	Anchor string
	// Raw suppresses the check that refuses a counted body line which parses as
	// a valid header. It is the escape hatch for the one thing body= exists to
	// make possible and the overcount check would otherwise forbid: writing a
	// REAL hunk header as content, which any plan editing this project's own
	// documentation or test fixtures has to do.
	Raw bool

	// SrcLine is the plan's own line number, for diagnostics.
	SrcLine int
	// Index is the hunk's position in the plan, used to order edits that begin
	// at the same line.
	Index int
}

// Parse reads a plan document. It returns every syntax error it found rather
// than the first, because a caller that has to re-emit the plan should learn
// about all of its mistakes in one round trip.
func Parse(r io.Reader) ([]Hunk, error) {
	var (
		hunks []Hunk
		errs  []string
		cur   *Hunk
		body  []string
		want  int  // remaining explicit body lines, -1 when scanning to next @@
		fixed bool // whether this hunk declared body=N at all
		stray bool // whether this hunk already reported an unaccounted line
	)
	flush := func() {
		if cur == nil {
			return
		}
		// body=N is a count, and a count the document does not honour means the
		// caller's picture of their own plan is wrong — the same class of
		// mistake as a drifted line number, which this format refuses.
		if fixed && want > 0 {
			errs = append(errs, fmt.Sprintf("line %d: body= asked for %d more line(s) than the plan contains",
				cur.SrcLine, want))
		}
		cur.Body = body
		hunks = append(hunks, *cur)
		cur, body, want, fixed, stray = nil, nil, -1, false, false
	}

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for n := 1; sc.Scan(); n++ {
		line := sc.Text()
		// Strip a UTF-8 BOM before deciding whether this line is a HEADER, on
		// EVERY line — not just the first.
		//
		// Windows PowerShell 5.1, the powershell.exe that ships with Windows,
		// writes a BOM for `-Encoding utf8` and has no BOM-less option
		// (utf8NoBOM arrived in PowerShell 7). So the obvious way to author a
		// plan there produced a file mrw refused, with a message that said
		// "text before the first @@ header" about a line that IS a header.
		//
		// ⚠ THE FIRST VERSION OF THIS FIX STRIPPED ONLY LINE 1, AND THAT WAS A
		// SILENT WRONG-WRITE. A PowerShell user builds a plan by concatenating
		// fragments, and every fragment carries its own BOM — so the SECOND
		// header arrived as "<BOM>@@ f.txt 3 replace", was not recognised as a
		// header, and became BODY TEXT of the first hunk. Two hunks applied as
		// one, at exit 0, writing the swallowed header into the caller's source
		// file. The same input was REFUSED before that fix: it made a loud
		// failure into a quiet corruption, which is the exact defect class this
		// format exists to prevent. Found in review, 2026-09-03.
		//
		// The strip is for the HEADER TEST ONLY. `line` keeps its original
		// bytes, so a BOM inside a body is still content and is written back
		// verbatim — stripping there would silently edit the caller's text,
		// which is the same sin one layer down. A body line that IS a valid
		// header after the strip is treated exactly as an un-BOM'd one would
		// be: as a header, unless raw= or a body= count says otherwise.
		hdr := strings.TrimPrefix(line, "\ufeff")
		// An explicit body= count takes precedence over header detection, so a
		// body may contain lines that themselves start with "@@ ".
		if cur != nil && want > 0 {
			// An OVERCOUNTED body= is the last silent way to lose a hunk: the
			// count runs past this hunk's text and eats the next header and
			// its body as content, after which the plan applies — correctly by
			// its own rules — missing an edit nobody will notice.
			//
			// A body line is refused only when it is a COMPLETE, VALID header,
			// so prose about the format still passes through, and `raw=true`
			// turns the check off for a body that means to contain one. The
			// narrower rule of "fire only when the count runs past the end of
			// the document" was weighed and rejected: the incident that
			// produced this check did not run past the end — the count landed
			// exactly on a header boundary and the plan parsed clean.
			if strings.HasPrefix(hdr, "@@ ") && !cur.Raw {
				if _, _, err := parseHeader(hdr, n); err == nil {
					errs = append(errs, fmt.Sprintf("line %d: body= still owes %d line(s), but %q is a "+
						"valid header — an overcounted body= would swallow that hunk. If the body really "+
						"contains a header, say raw=true", n, want, line))
				}
			}
			body = append(body, line)
			want--
			continue
		}
		// An exhausted count means the hunk is complete. Anything before the
		// next header is text the caller did not account for; absorbing it
		// silently is what made body=0 mean "unbounded" instead of "empty".
		if cur != nil && fixed && want == 0 && !strings.HasPrefix(hdr, "@@ ") {
			// Reported once per hunk, not once per line: a satisfied count
			// followed by a hundred lines is one mistake, and a hundred
			// identical errors would bury the other hunks' diagnostics.
			if t := strings.TrimSpace(line); t != "" && !stray {
				errs = append(errs, fmt.Sprintf("line %d: body= is satisfied; %q is not part of any hunk "+
					"(further lines before the next @@ header are not reported)", n, line))
				stray = true
			}
			continue
		}
		if !strings.HasPrefix(hdr, "@@ ") {
			if cur != nil {
				body = append(body, line)
				continue
			}
			if t := strings.TrimSpace(line); t != "" && !strings.HasPrefix(t, "#") {
				errs = append(errs, fmt.Sprintf("line %d: text before the first @@ header: %q", n, line))
			}
			continue
		}

		flush()
		h, explicit, err := parseHeader(hdr, n)
		if err != nil {
			errs = append(errs, fmt.Sprintf("line %d: %v", n, err))
			continue
		}
		h.Index = len(hunks)
		cur, body, want, fixed, stray = &h, nil, explicit, explicit >= 0, false
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("reading plan: %w", err)
	}
	flush()

	for i := range hunks {
		if err := validate(&hunks[i]); err != nil {
			errs = append(errs, fmt.Sprintf("line %d: %v", hunks[i].SrcLine, err))
		}
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("plan has %d error(s):\n  %s", len(errs), strings.Join(errs, "\n  "))
	}
	if len(hunks) == 0 {
		return nil, fmt.Errorf("plan is empty: no @@ headers found")
	}
	return hunks, nil
}

// parseHeader splits an "@@ path addr op key=value..." line. It returns the
// explicit body line count (-1 when the body runs to the next header).
func parseHeader(line string, srcLine int) (Hunk, int, error) {
	fields, err := splitHeader(strings.TrimPrefix(line, "@@ "))
	if err != nil {
		return Hunk{}, 0, err
	}
	if len(fields) < 3 {
		return Hunk{}, 0, fmt.Errorf("header needs at least <path> <addr> <op>, got %d field(s)", len(fields))
	}

	h := Hunk{Path: fields[0], Op: Op(fields[2]), Lines: -1, SrcLine: srcLine}
	switch h.Op {
	case OpReplace, OpInsertAfter, OpInsertBefore, OpDelete, OpCreate:
	default:
		return Hunk{}, 0, fmt.Errorf("unknown op %q (want replace, insert-after, insert-before, delete or create)", fields[2])
	}
	if h.Addr, err = ParseAddr(fields[1]); err != nil {
		return Hunk{}, 0, err
	}

	explicit := -1
	// A REPEATED key was last-wins and silent, which is this file's own stated
	// principle inverted: apply.go says "a guard that is parsed and then
	// discarded would be worse than no guard at all — the caller believes the
	// edit is pinned". A hunk written `anchor="NOPE" anchor="a"` applied at
	// exit 0 with the false guard gone (probed 2026-09-01, still true
	// 2026-09-03). Every key is affected, sha= and lines= included.
	//
	// Refused rather than resolved, because there is no correct winner: two
	// guards on one hunk are two different claims about the same edit, and
	// picking either one silently is how the caller keeps believing the other.
	seen := map[string]bool{}
	for _, opt := range fields[3:] {
		k, v, ok := strings.Cut(opt, "=")
		if !ok {
			return Hunk{}, 0, fmt.Errorf("option %q is not key=value", opt)
		}
		if seen[k] {
			return Hunk{}, 0, fmt.Errorf("%s= given twice: two guards on one hunk are two "+
				"different claims about the same edit, and mrw will not pick one for you", k)
		}
		seen[k] = true
		switch k {
		case "sha":
			// Length AND alphabet. The message has always promised "hex", and
			// checking only the length let sha=zzzzzzzz through to fail later
			// as a content mismatch — which sends the caller looking at their
			// file instead of at their plan (probed 2026-09-01).
			if len(v) < 8 {
				return Hunk{}, 0, fmt.Errorf("sha= needs at least 8 hex characters, got %q", v)
			}
			if strings.TrimLeft(strings.ToLower(v), "0123456789abcdef") != "" {
				return Hunk{}, 0, fmt.Errorf("sha= is not hexadecimal: %q", v)
			}
			h.SHA = strings.ToLower(v)
		case "lines":
			if h.Lines, err = strconv.Atoi(v); err != nil || h.Lines < 0 {
				return Hunk{}, 0, fmt.Errorf("lines= wants a non-negative integer, got %q", v)
			}
		case "anchor":
			// An empty anchor asserts nothing while reading exactly like an
			// assertion, and apply reads an empty Anchor as "no anchor given".
			if v == "" {
				return Hunk{}, 0, fmt.Errorf("anchor= needs a value")
			}
			h.Anchor = v
		case "raw":
			if v != "true" {
				return Hunk{}, 0, fmt.Errorf("raw= takes only %q, got %q", "true", v)
			}
			h.Raw = true
		case "body":
			if explicit, err = strconv.Atoi(v); err != nil || explicit < 0 {
				return Hunk{}, 0, fmt.Errorf("body= wants a non-negative integer, got %q", v)
			}
		default:
			return Hunk{}, 0, fmt.Errorf("unknown option %q (want sha, lines, anchor, body or raw)", k)
		}
	}
	// raw= switches off the valid-header check INSIDE a counted body, so
	// without body= there is no body for it to act on and the caller has
	// written a guard that cannot fire — the same class as the duplicate key
	// above, and refused for the same reason.
	if h.Raw && explicit < 0 {
		return Hunk{}, 0, fmt.Errorf("raw=true without body=: raw= only switches off the header " +
			"check inside a counted body, so on its own it guards nothing")
	}
	return h, explicit, nil
}

// splitHeader splits on whitespace but keeps double-quoted fields together, so
// a path with a space or an anchor= holding a phrase survives.
func splitHeader(s string) ([]string, error) {
	var (
		out   []string
		cur   strings.Builder
		inTok bool
		inQ   bool
		// inPat is ADR-013's pattern address. A regex is a single token even
		// though it contains spaces — `/^func (s *Store) Get/` is the whole
		// point of the feature and it has two. Without this the header splits
		// mid-pattern and the op is reported as `\(s`, which is what the
		// end-to-end run reported before this existed: the unit tests passed
		// because they called ParseAddr directly and never went through here.
		inPat bool
	)
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch {
		case inPat && r == '\\' && i+1 < len(rs) && rs[i+1] == '/':
			// \/ is a literal slash inside a pattern, not its terminator.
			cur.WriteRune(r)
			cur.WriteRune(rs[i+1])
			i++
		case inPat && r == '/':
			cur.WriteRune(r)
			// `/a/,/b/` is one address: swallow the comma and keep scanning.
			if i+2 < len(rs) && rs[i+1] == ',' && rs[i+2] == '/' {
				cur.WriteRune(',')
				cur.WriteRune('/')
				i += 2
			} else {
				inPat = false
			}
		case r == '/' && !inTok && !inQ:
			inPat, inTok = true, true
			cur.WriteRune(r)
		case r == '\\' && i+1 < len(rs) && (rs[i+1] == '"' || rs[i+1] == '\\'):
			// A backslash escapes a quote or another backslash, so an anchor
			// can name code that itself contains a quote. Without this the
			// quote toggles the quoting state and the backslash survives into
			// the value, producing a guard that matches nothing.
			cur.WriteRune(rs[i+1])
			inTok = true
			i++
		case r == '"':
			inQ, inTok = !inQ, true
		case (r == ' ' || r == '\t') && !inQ && !inPat:
			if inTok {
				out = append(out, cur.String())
				cur.Reset()
				inTok = false
			}
		default:
			cur.WriteRune(r)
			inTok = true
		}
	}
	if inQ {
		return nil, fmt.Errorf("unterminated quote in header")
	}
	if inTok {
		out = append(out, cur.String())
	}
	return out, nil
}

// parsePattern reads the `/re/` and `/re/,/re/` forms. The syntax is the one
// internal/read already accepts, so a caller learns one address language
// rather than two — but the RULE differs, and deliberately: read serving two
// matches is useful, apply editing two matches is a bug. ADR-013's exactly-once
// rule lives in the resolver, not here.
func parsePattern(s string) (Addr, error) {
	// Scan to the closing slash, honouring \/ so a pattern may contain one.
	end := -1
	for i := 1; i < len(s); i++ {
		if s[i] == '/' && s[i-1] != '\\' {
			end = i
			break
		}
	}
	if end < 0 {
		return Addr{}, fmt.Errorf("pattern %q is never closed: expected a second /", s)
	}
	body := s[1:end]
	rest := s[end+1:]
	start, err := regexp.Compile(body)
	if err != nil {
		return Addr{}, fmt.Errorf("bad pattern %q: %w", body, err)
	}
	if body == "" {
		return Addr{}, fmt.Errorf("empty pattern // matches every line, so it can never resolve to exactly one")
	}
	a := Addr{StartPat: start}
	switch {
	case rest == "":
		return a, nil
	case strings.HasPrefix(rest, ",/"):
		tail, err := parsePattern(rest[1:])
		if err != nil {
			return Addr{}, err
		}
		if tail.EndPat != nil {
			return Addr{}, fmt.Errorf("address %q has more than two endpoints", s)
		}
		a.EndPat = tail.StartPat
		return a, nil
	default:
		return Addr{}, fmt.Errorf("unexpected %q after the pattern in %q", rest, s)
	}
}

// ParseAddr reads an address: "N", "N-M", "N-" (to end of file), "$" (last
// line), "0" (before the first line) or "-" (no address, for create).
func ParseAddr(s string) (Addr, error) {
	switch s {
	case "":
		return Addr{}, fmt.Errorf("empty address")
	case "-":
		return Addr{Start: 0, End: 0}, nil
	case "$":
		return Addr{Start: EOF, End: EOF}, nil
	}
	// A pattern is scanned before anything splits on "-" or ",": those are
	// ordinary characters inside a regex, and cutting first is how `/a-b/`
	// becomes a bad line number.
	if strings.HasPrefix(s, "/") {
		return parsePattern(s)
	}
	one := func(t string) (int, error) {
		if t == "$" {
			return EOF, nil
		}
		n, err := strconv.Atoi(t)
		if err != nil || n < 0 {
			return 0, fmt.Errorf("bad line number %q", t)
		}
		return n, nil
	}
	lo, hi, ranged := strings.Cut(s, "-")
	start, err := one(lo)
	if err != nil {
		return Addr{}, err
	}
	if !ranged {
		return Addr{Start: start, End: start}, nil
	}
	if hi == "" { // "N-" means N to end of file
		return Addr{Start: start, End: EOF}, nil
	}
	end, err := one(hi)
	if err != nil {
		return Addr{}, err
	}
	return Addr{Start: start, End: end}, nil
}

// validate checks the parts of a hunk that need no file on disk: op/address
// agreement and whether a body is meaningful for the op.
func validate(h *Hunk) error {
	// A pattern address has no line bounds yet — internal/apply resolves it
	// against the file, which is the only place the file's length is known. So
	// the checks below that reason about Start and End cannot speak about it,
	// and `patterned` is how they say so rather than reading a zero as an
	// address. Everything that does NOT depend on the resolved lines — an empty
	// body, a create with a guard — still applies.
	patterned := h.Addr.StartPat != nil
	switch h.Op {
	case OpDelete:
		// A body on a delete used to be a hard parse error. It now means "these
		// are the lines I expect to remove" — the caller's own picture of the
		// range, which is the one assertion mrw cannot derive for them. Whether
		// it HOLDS needs the file, so it is checked in internal/apply; there is
		// nothing to check here (ADR-008).
	case OpCreate:
		// A pattern IS an address, so `create` refuses it exactly as it refuses
		// a line number. Gating this on !patterned let `@@ new.go /x/ create`
		// through with `ok` while `@@ new.go 1 create` was refused — two
		// address forms in one grammar have to be refused on the same inputs.
		// Caught in review of PR #74.
		if h.Addr.StartPat != nil {
			return fmt.Errorf("create takes no address, use %q", "-")
		}
		if !patterned && (h.Addr.Start != 0 || h.Addr.End != 0) {
			return fmt.Errorf("create takes no address, use %q", "-")
		}
		if h.Anchor != "" || h.Lines >= 0 {
			return fmt.Errorf("create takes no anchor= or lines= (the file must not exist yet)")
		}
	case OpInsertAfter, OpInsertBefore:
		// A RANGE is a range whether it is written 3-6 or /a/,/b/, and an
		// insertion addresses one line. Whether the two patterns happen to
		// resolve to the same line is not knowable here and does not matter:
		// the caller wrote a range.
		if h.Addr.EndPat != nil {
			return fmt.Errorf("%s takes a single line, not a range", h.Op)
		}
		if !patterned && h.Addr.Start != h.Addr.End {
			return fmt.Errorf("%s takes a single line, not the range %s", h.Op, h.Addr)
		}
		if len(h.Body) == 0 {
			return fmt.Errorf("%s with an empty body would change nothing", h.Op)
		}
	case OpReplace:
		if !patterned && h.Addr.Start == 0 {
			return fmt.Errorf("replace needs a real line range, got %s", h.Addr)
		}
		// A replace with no body DELETES the addressed lines while reporting
		// "ok". A plan whose body was lost in transit — a truncated emission,
		// an editor eating the last line — would remove code and hand back a
		// receipt saying it succeeded, which is the failure this whole format
		// exists to refuse. The mirror image was already policed: `delete` with
		// a body is an error. Nothing is lost by refusing this one, because
		// deleting lines is what `delete` is for.
		if len(h.Body) == 0 {
			return fmt.Errorf("replace with an empty body would delete %s — say delete if that is "+
				"what you mean, and check the body did not go missing if it is not", h.Addr)
		}
	}
	if h.Op != OpInsertBefore && h.Addr.Start != EOF && h.Addr.End != EOF &&
		h.Addr.End < h.Addr.Start {
		return fmt.Errorf("address %s ends before it starts", h.Addr)
	}
	return nil
}
