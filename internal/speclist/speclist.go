// Package speclist reads a list of read specs, one a line, the way
// `mrw read --files-from` and mrw_read's files_from both take one (ADR-117).
package speclist

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// MaxLine is the longest one spec in a list may be.
const MaxLine = 8 * 1024 * 1024

// mark is the UTF-8 byte order mark.
const mark = "\xef\xbb\xbf"

// Parse reads the specs in r. A blank line, or one whose first non-space
// character is "#", is skipped; every other line is a spec as written, edge
// spaces kept (ADR-069), except that the UTF-8 byte order marks at the head
// of the first line are dropped (ADR-117 T2, T3). label names the list in
// every error, so each surface keeps its own word for it ("--files-from x",
// "files_from x"). A list that holds no spec is an error, never an empty
// read. notes, when not nil, takes a line about what Parse changed in the
// list as written.
func Parse(r io.Reader, label string, notes io.Writer) ([]string, error) {
	var out []string
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), MaxLine)
	n := 0
	for sc.Scan() {
		n++
		line := sc.Text()
		// A list saved by Notepad, or piped by PowerShell with a UTF-8
		// preamble, starts with a byte order mark that no spec means (ADR-117
		// T2); Windows PowerShell 5.1 can pipe two, so every one goes (T3).
		if n == 1 {
			marks := 0
			for strings.HasPrefix(line, mark) {
				line, marks = line[len(mark):], marks+1
			}
			if notes != nil && marks == 1 {
				_, _ = fmt.Fprintf(notes, "mrw: %s: dropped the byte order mark at the start of the list\n", label)
			}
			if notes != nil && marks > 1 {
				_, _ = fmt.Fprintf(notes, "mrw: %s: dropped %d byte order marks at the start of the list\n", label, marks)
			}
		}
		if t := strings.TrimSpace(line); t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		out = append(out, line)
	}
	if err := sc.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			// The scanner stops AT the long line and never returns it, so the
			// line it failed on is the one after the last it counted (ADR-074).
			return nil, fmt.Errorf("%s line %d: longer than the 8 MiB a spec may be", label, n+1)
		}
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s: no specs (blank lines and # comments are skipped)", label)
	}
	return out, nil
}
