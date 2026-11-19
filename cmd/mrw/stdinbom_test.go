package main

import (
	"strings"
	"testing"
)

// A UTF-8 byte order mark at the very start of standard input is not part of
// line 1. PowerShell puts one there when $OutputEncoding is UTF-8 with a
// preamble, and cmd.exe's `type` passes a file's own mark through; `mrw read
// --grep '^one' -` then finds no line, because line 1 is "\ufeffone" (found by
// a Windows peer session, 2026-11-19). A read of standard input records and
// licenses nothing (ADR-161), so dropping the mark changes no licence; a read
// of a FILE serves the mark as it always has.
func TestAByteOrderMarkOnStandardInputIsNotPartOfLineOne(t *testing.T) {
	run, _ := ifChangedRun(t, map[string]string{"f.txt": "\xef\xbb\xbfone\ntwo\n"})
	const bom = "\xef\xbb\xbf"

	// The symptom: an anchored pattern finds line 1, and what is served holds no mark.
	out, errOut, code := run(bom+"one\ntwo\n", "read", "--grep", "^one", "-")
	if code != 0 || !strings.Contains(out, " 1| one\n") || strings.Contains(out, bom) {
		t.Errorf("--grep ^one on marked input: exit %d, want line 1 served without the mark:\n%q\n%s", code, out, errOut)
	}
	if !strings.Contains(errOut, "byte order mark") {
		t.Errorf("the dropped mark is not named on standard error:\n%s", errOut)
	}
	// The header describes what is served: 2 lines, 8 bytes, not 11.
	if !strings.Contains(out, "==> -  2L  8B") {
		t.Errorf("the header counts the mark:\n%q", out)
	}

	// A plain read and an inverted read see the same line 1.
	if out, _, code := run(bom+"one\ntwo\n", "read", "-"); code != 0 || !strings.Contains(out, " 1| one\n") || strings.Contains(out, bom) {
		t.Errorf("read - on marked input: exit %d:\n%q", code, out)
	}
	if out, _, code := run(bom+"one\ntwo\n", "read", "--grep", "two", "--invert", "-"); code != 0 || !strings.Contains(out, " 1| one\n") || strings.Contains(out, bom) {
		t.Errorf("--invert on marked input: exit %d:\n%q", code, out)
	}

	// Several marks at the very start all go, and the note says how many.
	if out, errOut, code := run(bom+bom+"one\n", "read", "-"); code != 0 || strings.Contains(out, bom) || !strings.Contains(errOut, "2 ") {
		t.Errorf("two leading marks: exit %d:\n%q\n%s", code, out, errOut)
	}

	// Only LEADING marks go. With a mark at the start and another at the start of
	// line 2, an implementation that strips every mark once it finds a leading one
	// would pass the leading-only and interior-only cases below; here it serves
	// 8 bytes and loses line 2's mark (found by a pre-code challenge).
	if out, errOut, code := run(bom+"one\n"+bom+"two\n", "read", "-"); code != 0 || !strings.Contains(out, "==> -  2L  11B") ||
		!strings.Contains(out, " 1| one\n") || !strings.Contains(out, " 2| "+bom+"two\n") || !strings.Contains(errOut, "1 ") || strings.Contains(errOut, "2 ") {
		t.Errorf("a leading mark and one on line 2: exit %d, want 2L 11B, line 1 exactly one, line 2 keeping its mark, and a note of one mark:\n%q\n%s", code, out, errOut)
	}

	// Controls that must not change.
	// 1. No mark: no note, and line 1 as today.
	if out, errOut, code := run("one\ntwo\n", "read", "-"); code != 0 || strings.Contains(errOut, "byte order mark") || !strings.Contains(out, "==> -  2L  8B") {
		t.Errorf("unmarked input: exit %d:\n%q\n%s", code, out, errOut)
	}
	// 2. A mark that is not at the very start is content and is served.
	if out, errOut, code := run("one\n"+bom+"two\n", "read", "-"); code != 0 || !strings.Contains(out, bom+"two") || strings.Contains(errOut, "byte order mark") {
		t.Errorf("a mark at the start of line 2: exit %d:\n%q\n%s", code, out, errOut)
	}
	// 3. A file keeps its mark: a read of a file serves the bytes it holds.
	if out, _, code := run("", "read", "f.txt"); code != 0 || !strings.Contains(out, bom+"one") {
		t.Errorf("a file's mark was dropped: exit %d:\n%q", code, out)
	}
	// 4. Input that is only a mark is empty input, as it is today without one.
	if _, errOut, code := run(bom, "read", "--grep", "x", "-"); code != 1 || !strings.Contains(errOut, "no match") {
		t.Errorf("only a mark: exit %d:\n%s", code, errOut)
	}
}
