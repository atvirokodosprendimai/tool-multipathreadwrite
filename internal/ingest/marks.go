package ingest

import (
	"bufio"
	"bytes"
	"fmt"
)

// LeadingMarks counts the UTF-8 byte order marks at the head of r without
// consuming any, up to what r's buffer holds. plan.Parse drops every one of
// them (issue #537); the caller is told when there were several.
func LeadingMarks(r *bufio.Reader) int {
	n := 0
	for {
		b, _ := r.Peek(3 * (n + 1))
		if !bytes.HasPrefix(b[3*n:], []byte("\xef\xbb\xbf")) {
			return n
		}
		n++
	}
}

// PlanMarksNote is the one-line note, without the "mrw:" prefix, for a plan
// that began with n UTF-8 byte order marks, n above one. A single mark is
// silent: it is what PowerShell has always added (create142_test.go).
func PlanMarksNote(n int) string {
	return fmt.Sprintf("the plan begins with %d UTF-8 byte order marks, all dropped (Windows PowerShell 5.1 pipes two when $OutputEncoding is UTF-8 with a preamble and [Console]::InputEncoding is too, as in a shell started under code page 65001)", n)
}
