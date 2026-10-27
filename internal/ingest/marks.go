package ingest

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"strings"
)

// utf8Mark is the UTF-8 byte order mark.
const utf8Mark = "\xef\xbb\xbf"

// LeadingMarks counts the UTF-8 byte order marks at the head of r without
// consuming any. plan.Parse drops every one (issue #537); the caller is told
// when there were several. The count stops at r's buffer (1365 marks of a
// 4096-byte one): more reports that it did, so the number is a floor, not the
// whole.
func LeadingMarks(r *bufio.Reader) (n int, more bool) {
	for {
		b, err := r.Peek(3 * (n + 1))
		if bytes.HasPrefix(b[3*n:], []byte(utf8Mark)) {
			n++
			continue
		}
		return n, errors.Is(err, bufio.ErrBufferFull) && strings.HasPrefix(utf8Mark, string(b[3*n:]))
	}
}

// DropLeadingMarks returns doc without the UTF-8 byte order marks at its head,
// and how many there were. The foreign formats read their documents through it
// (review of #539): Windows PowerShell 5.1 pipes one or two before them.
func DropLeadingMarks(doc []byte) ([]byte, int) {
	n := 0
	for bytes.HasPrefix(doc, []byte(utf8Mark)) {
		doc = doc[3:]
		n++
	}
	return doc, n
}

// MarksNote is the one-line note, without the "mrw:" prefix, for a document
// (what: "the plan", "the apply_patch document") that began with n UTF-8 byte
// order marks, n above one; more says the count is a floor. A single mark is
// silent: it is what PowerShell has always added (create142_test.go). The
// parenthesis claims only what the Windows probes measured (2026-10-27).
func MarksNote(what string, n int, more bool) string {
	count := fmt.Sprint(n)
	if more {
		count += " or more"
	}
	return fmt.Sprintf("%s begins with %s UTF-8 byte order marks, all dropped (measured: Windows PowerShell 5.1 pipes two when $OutputEncoding is UTF8Encoding($true), in a console under code page 65001)", what, count)
}
