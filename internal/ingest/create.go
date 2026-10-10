package ingest

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/lines"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted"
)

// maxPlanLine is plan.Parse's scanner buffer: a line and its terminator must fit.
const maxPlanLine = 16 * 1024 * 1024

// CheckCreatePath refuses a --create PATH no plan could name: empty, not
// relative to the root, or holding a newline or NUL, which a header line cannot
// carry. It is asked before standard input is read, so a usage error does not
// wait for EOF (the Codex review of #371).
func CheckCreatePath(path string) error {
	if path == "" {
		return fmt.Errorf("--create needs a PATH")
	}
	if rooted.IsRooted(path) {
		return fmt.Errorf("--create %s: the path is not relative to the root", path)
	}
	if strings.ContainsAny(path, "\n\r\x00") {
		return fmt.Errorf("--create %q: a path cannot hold a newline or a NUL", path)
	}
	return nil
}

// CreateContent cleans what a pipe added to the caller's content, and names
// each change (ADR-142). Windows PowerShell and PowerShell 7 append CRLF to the
// text they pipe, so an LF file arrives as "a\nb\n\r\n", which CompileCreate
// refuses as a line ending in a bare CR. A final CRLF after text that holds an
// LF and no other CR is that terminator and is dropped; every other shape is
// returned as it came, for CompileCreate to accept or refuse. A leading UTF-8
// byte order mark is content and is kept, but PowerShell 5.1 adds one, so it is
// named. Each note is one line, without the "mrw:" prefix.
func CreateContent(raw []byte) ([]byte, []string) {
	var notes []string
	if bytes.HasPrefix(raw, []byte("\xef\xbb\xbf")) {
		notes = append(notes, "the content begins with a UTF-8 byte order mark, written to the file as given (Windows PowerShell 5.1 adds one to what it pipes)")
	}
	if body, ok := bytes.CutSuffix(raw, []byte("\r\n")); ok && bytes.IndexByte(body, '\n') >= 0 && bytes.IndexByte(body, '\r') < 0 {
		notes = append(notes, "the content ended in CRLF after lines that end in LF; that CRLF, which PowerShell appends to what it pipes, was dropped")
		raw = body
	}
	return raw, notes
}

// CompileCreate turns a file's content into the native plan text that creates
// path, so `mrw write --create PATH` is a plan the caller did not have to count
// (ADR-139). The hunk always declares `body=N raw=true`: the parser then takes
// exactly N lines as content whatever they look like, so a content line that is a
// header (a BOM-prefixed one included) cannot start another hunk. The path is
// always double-quoted and escaped, so no name is read as other syntax.
//
// The file is what a create plan makes of the content's lines: each ends in a
// newline, so a missing final newline is added and CRLF becomes LF, and empty
// content makes an empty file. Content mrw cannot split into lines, or whose
// lines end in a bare CR (mixed line endings, which the plan parser would strip
// without a word), is refused. The function writes nothing.
func CompileCreate(path string, content []byte) ([]byte, error) {
	if err := CheckCreatePath(path); err != nil {
		return nil, err
	}
	if why := lines.Unsplittable(content); why != "" {
		return nil, fmt.Errorf("--create %s: the content %s, so mrw cannot carry it as lines", path, why)
	}
	ls, _, _ := lines.Split(string(content))
	for _, l := range ls {
		// Whatever terminator Split chose, a line that still ends in CR (mixed
		// endings, or a CR left by the last line of CRLF content) would have it
		// stripped by the plan parser's line scanner, quietly.
		if strings.HasSuffix(l, "\r") {
			return nil, fmt.Errorf("--create %s: a line ends in a bare CR (mixed line endings), which a plan cannot carry; make every ending LF or every ending CRLF", path)
		}
		// The parser reads a plan through a bounded scanner; a longer line would
		// fail there and be counted as a plan that did not parse (ADR-009), though
		// the caller's content was the problem.
		if len(l)+2 > maxPlanLine {
			return nil, fmt.Errorf("--create %s: a line is %d bytes, longer than the %d a plan carries", path, len(l), maxPlanLine-2)
		}
	}
	quoted := strings.ReplaceAll(path, `\`, `\\`)
	quoted = strings.ReplaceAll(quoted, `"`, `\"`)
	var b strings.Builder
	fmt.Fprintf(&b, "@@ \"%s\" 0 create body=%d raw=true\n", quoted, len(ls))
	for _, l := range ls {
		b.WriteString(l)
		b.WriteByte('\n')
	}
	return []byte(b.String()), nil
}
