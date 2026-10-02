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

// Parse reads the specs in r. A blank line, or one whose first non-space
// character is "#", is skipped; every other line is a spec as written, edge
// spaces kept (ADR-069). label names the list in every error, so each surface
// keeps its own word for it ("--files-from x", "files_from x"). A list that
// holds no spec is an error, never an empty read.
func Parse(r io.Reader, label string) ([]string, error) {
	var out []string
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), MaxLine)
	n := 0
	for sc.Scan() {
		n++
		line := sc.Text()
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
