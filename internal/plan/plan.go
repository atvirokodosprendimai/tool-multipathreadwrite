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

// Addr is an inclusive 1-based line range. Start may be 0 for the position
// before the first line, and either bound may be EOF.
type Addr struct {
	Start int
	End   int
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
		want  int // remaining explicit body lines, -1 when scanning to next @@
	)
	flush := func() {
		if cur == nil {
			return
		}
		cur.Body = body
		hunks = append(hunks, *cur)
		cur, body, want = nil, nil, -1
	}

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for n := 1; sc.Scan(); n++ {
		line := sc.Text()

		// An explicit body= count takes precedence over header detection, so a
		// body may contain lines that themselves start with "@@ ".
		if cur != nil && want > 0 {
			body = append(body, line)
			want--
			continue
		}
		if !strings.HasPrefix(line, "@@ ") {
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
		h, explicit, err := parseHeader(line, n)
		if err != nil {
			errs = append(errs, fmt.Sprintf("line %d: %v", n, err))
			continue
		}
		h.Index = len(hunks)
		cur, body, want = &h, nil, explicit
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
	for _, opt := range fields[3:] {
		k, v, ok := strings.Cut(opt, "=")
		if !ok {
			return Hunk{}, 0, fmt.Errorf("option %q is not key=value", opt)
		}
		switch k {
		case "sha":
			if len(v) < 8 {
				return Hunk{}, 0, fmt.Errorf("sha= needs at least 8 hex characters, got %q", v)
			}
			h.SHA = strings.ToLower(v)
		case "lines":
			if h.Lines, err = strconv.Atoi(v); err != nil || h.Lines < 0 {
				return Hunk{}, 0, fmt.Errorf("lines= wants a non-negative integer, got %q", v)
			}
		case "anchor":
			h.Anchor = v
		case "body":
			if explicit, err = strconv.Atoi(v); err != nil || explicit < 0 {
				return Hunk{}, 0, fmt.Errorf("body= wants a non-negative integer, got %q", v)
			}
		default:
			return Hunk{}, 0, fmt.Errorf("unknown option %q (want sha, lines, anchor or body)", k)
		}
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
	)
	for _, r := range s {
		switch {
		case r == '"':
			inQ, inTok = !inQ, true
		case (r == ' ' || r == '\t') && !inQ:
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
	switch h.Op {
	case OpDelete:
		if len(h.Body) > 0 {
			return fmt.Errorf("delete takes no body, got %d line(s)", len(h.Body))
		}
	case OpCreate:
		if h.Addr.Start != 0 || h.Addr.End != 0 {
			return fmt.Errorf("create takes no address, use %q", "-")
		}
		if h.Anchor != "" || h.Lines >= 0 {
			return fmt.Errorf("create takes no anchor= or lines= (the file must not exist yet)")
		}
	case OpInsertAfter, OpInsertBefore:
		if h.Addr.Start != h.Addr.End {
			return fmt.Errorf("%s takes a single line, not the range %s", h.Op, h.Addr)
		}
		if len(h.Body) == 0 {
			return fmt.Errorf("%s with an empty body would change nothing", h.Op)
		}
	case OpReplace:
		if h.Addr.Start == 0 {
			return fmt.Errorf("replace needs a real line range, got %s", h.Addr)
		}
	}
	if h.Op != OpInsertBefore && h.Addr.Start != EOF && h.Addr.End != EOF &&
		h.Addr.End < h.Addr.Start {
		return fmt.Errorf("address %s ends before it starts", h.Addr)
	}
	return nil
}
