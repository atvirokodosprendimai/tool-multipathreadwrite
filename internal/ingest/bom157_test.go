package ingest

import (
	"bytes"
	"strings"
	"testing"
)

// ADR-157 Decision 7. Windows PowerShell 5.1 under code page 65001 with a
// $OutputEncoding that has a preamble pipes two byte order marks; a create
// keeps every one, as ADR-142 keeps one, and its note counts them. One keeps
// the sentence it had.
func TestACreateNamesEveryLeadingByteOrderMark(t *testing.T) {
	const bom = "\xef\xbb\xbf"
	for n, want := range map[int]string{
		1: "the content begins with a UTF-8 byte order mark, kept as content",
		2: "the content begins with 2 UTF-8 byte order marks, kept as content",
		3: "the content begins with 3 UTF-8 byte order marks, kept as content",
	} {
		raw := []byte(strings.Repeat(bom, n) + "one\ntwo\n")
		content, notes := CreateContent(raw)
		if !bytes.Equal(content, raw) {
			t.Errorf("%d mark(s): the content changed: %q", n, content)
		}
		if len(notes) != 1 || !strings.HasPrefix(notes[0], want) {
			t.Errorf("%d mark(s): notes %q, want one starting %q", n, notes, want)
		}
	}
}
