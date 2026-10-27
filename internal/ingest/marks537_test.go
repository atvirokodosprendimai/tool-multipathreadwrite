package ingest

import (
	"bufio"
	"strings"
	"testing"
)

// Issue #537. LeadingMarks counts the UTF-8 marks at the head of a plan
// without consuming them; the note is for more than one.
func TestLeadingMarksCountsEveryMarkAndConsumesNone(t *testing.T) {
	for n := 0; n <= 3; n++ {
		in := strings.Repeat(bom, n) + "@@ f 1 replace\nx\n"
		br := bufio.NewReader(strings.NewReader(in))
		if got := LeadingMarks(br); got != n {
			t.Errorf("%d mark(s): counted %d", n, got)
		}
		rest, _ := br.Peek(len(in))
		if string(rest) != in {
			t.Errorf("%d mark(s): the reader lost bytes: %q", n, rest)
		}
	}
	if got := LeadingMarks(bufio.NewReader(strings.NewReader("\xef\xbb"))); got != 0 {
		t.Errorf("a partial mark counted %d", got)
	}
	if got := LeadingMarks(bufio.NewReader(strings.NewReader(strings.Repeat(bom, 1000)))); got != 1000 {
		t.Errorf("1000 marks, within the reader's buffer: counted %d", got)
	}
}

// The apply_patch compiler asks the parser's question: a body line that is a
// header after every leading mark is dropped needs a count.
func TestAnApplyPatchBodyLineWithTwoMarksBeforeAHeaderIsCounted(t *testing.T) {
	got := emit("replace", "f.txt", 1, 1, []string{bom + bom + "@@ x 1 replace"}, "")
	if !strings.Contains(got, "raw=true") {
		t.Errorf("hunk %q lacks body= raw=true", got)
	}
}
