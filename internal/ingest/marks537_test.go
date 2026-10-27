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
		if got, more := LeadingMarks(br); got != n || more {
			t.Errorf("%d mark(s): counted %d, more %v", n, got, more)
		}
		rest, _ := br.Peek(len(in))
		if string(rest) != in {
			t.Errorf("%d mark(s): the reader lost bytes: %q", n, rest)
		}
	}
	if got, more := LeadingMarks(bufio.NewReader(strings.NewReader("\xef\xbb"))); got != 0 || more {
		t.Errorf("a partial mark counted %d, more %v", got, more)
	}
	if got, more := LeadingMarks(bufio.NewReader(strings.NewReader(strings.Repeat(bom, 1000)))); got != 1000 || more {
		t.Errorf("1000 marks, within the reader's buffer: counted %d, more %v", got, more)
	}
}

// The count stops at the reader's buffer (4096 bytes, 1365 marks); it says so
// rather than report a smaller number as the whole (review of #539).
func TestLeadingMarksSaysWhenTheBufferEndedTheCount(t *testing.T) {
	got, more := LeadingMarks(bufio.NewReader(strings.NewReader(strings.Repeat(bom, 2000) + "@@ f 1 replace\n")))
	if got != 1365 || !more {
		t.Errorf("2000 marks: counted %d, more %v, want 1365 and more", got, more)
	}
	if note := MarksNote("the plan", got, more); !strings.Contains(note, "begins with 1365 or more UTF-8 byte order marks") {
		t.Errorf("note %q does not say 1365 or more", note)
	}
	if note := MarksNote("the plan", 2, false); !strings.Contains(note, "begins with 2 UTF-8 byte order marks, all dropped") || strings.Contains(note, "or more") {
		t.Errorf("note %q", note)
	}
}

// The note claims what the Windows probes measured: UTF8Encoding($true) in a
// console under code page 65001, and nothing about [Console]::InputEncoding.
func TestTheMarksNoteClaimsOnlyWhatWasMeasured(t *testing.T) {
	note := MarksNote("the plan", 2, false)
	for _, want := range []string{"UTF8Encoding($true)", "code page 65001"} {
		if !strings.Contains(note, want) {
			t.Errorf("note %q lacks %q", note, want)
		}
	}
	if strings.Contains(note, "InputEncoding") {
		t.Errorf("note %q claims [Console]::InputEncoding, which the probes did not measure", note)
	}
}

// Review of #539. --format apply_patch and search_replace refused a single
// leading mark ("text before *** Begin Patch"; the mark read as part of the
// path), so Windows PowerShell 5.1 could pipe neither. Every leading mark is
// dropped, in the compiler, so the MCP surface gets it too.
func TestTheForeignFormatsDropEveryLeadingByteOrderMark(t *testing.T) {
	root := writeTree(t, map[string]string{"a.go": demo})
	const patch = "*** Begin Patch\n*** Add File: n.txt\n+hi\n*** End Patch\n"
	wantPatch, err := CompileApplyPatch(root, []byte(patch))
	if err != nil {
		t.Fatal(err)
	}
	wantSR, err := CompileSearchReplace(root, []byte(twoHunkSearchReplace))
	if err != nil {
		t.Fatal(err)
	}
	for n := 1; n <= 3; n++ {
		got, err := CompileApplyPatch(root, []byte(strings.Repeat(bom, n)+patch))
		if err != nil || string(got) != string(wantPatch) {
			t.Errorf("apply_patch, %d mark(s): %q, %v, want %q", n, got, err, wantPatch)
		}
		got, err = CompileSearchReplace(root, []byte(strings.Repeat(bom, n)+twoHunkSearchReplace))
		if err != nil || string(got) != string(wantSR) {
			t.Errorf("search_replace, %d mark(s): %q, %v, want %q", n, got, err, wantSR)
		}
	}
	// A mark that is not at the head is not dropped.
	if _, err := CompileApplyPatch(root, []byte("x\n"+bom+patch)); err == nil {
		t.Error("text before the patch with a mark after it was accepted")
	}
}

func TestDropLeadingMarksCountsAndKeepsTheRest(t *testing.T) {
	for n := 0; n <= 3; n++ {
		got, c := DropLeadingMarks([]byte(strings.Repeat(bom, n) + bom[:2] + "x" + bom))
		if c != n || string(got) != bom[:2]+"x"+bom {
			t.Errorf("%d mark(s): dropped %d, left %q", n, c, got)
		}
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
