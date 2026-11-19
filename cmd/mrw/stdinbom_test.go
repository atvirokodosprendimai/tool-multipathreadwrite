package main

import (
	"strings"
	"testing"
)

// A UTF-8 byte order mark at the very start of standard input is not part of
// line 1. Windows PowerShell 5.1 under code page 65001 puts one on every pipe
// (from [Console]::InputEncoding), a second when $OutputEncoding is UTF-8 with
// a preamble, and cmd.exe's `type` passes a file's own mark through; `mrw read
// --grep '^one' -` then found no line, because line 1 was "\ufeffone" (found
// by a Windows peer session, 2026-11-19). A read of standard input records and
// licenses nothing (ADR-161), so dropping the mark changes no licence; a read
// of a FILE serves the mark as it always has.
func TestAByteOrderMarkOnStandardInputIsNotPartOfLineOne(t *testing.T) {
	run, _ := ifChangedRun(t, map[string]string{"f.txt": "\xef\xbb\xbfone\ntwo\n"})
	const bom = "\xef\xbb\xbf"
	const note = "UTF-8 byte order marks, all dropped"

	// The symptom: an anchored pattern finds line 1, and what is served holds no
	// mark. One mark is silent, as it is on a plan: on those machines every pipe
	// carries it.
	out, errOut, code := run(bom+"one\ntwo\n", "read", "--grep", "^one", "-")
	if code != 0 || !strings.Contains(out, " 1| one\n") || strings.Contains(out, bom) {
		t.Errorf("--grep ^one on marked input: exit %d, want line 1 served without the mark:\n%q\n%s", code, out, errOut)
	}
	if errOut != "" {
		t.Errorf("one mark is not silent:\n%s", errOut)
	}
	// The header describes what is served: 2 lines, 8 bytes, not 11.
	if !strings.Contains(out, "==> -  2L  8B") {
		t.Errorf("the header counts the mark:\n%q", out)
	}

	// A plain read and an inverted read see the same line 1: inverted on ^one,
	// line 1 is left out only if the matcher saw it without its mark.
	if out, _, code := run(bom+"one\ntwo\n", "read", "-"); code != 0 || !strings.Contains(out, " 1| one\n") || strings.Contains(out, bom) {
		t.Errorf("read - on marked input: exit %d:\n%q", code, out)
	}
	if out, _, code := run(bom+"one\ntwo\n", "read", "--grep", "^one", "--invert", "-"); code != 0 || strings.Contains(out, " 1| ") || !strings.Contains(out, " 2| two\n") {
		t.Errorf("--invert ^one on marked input: exit %d, want line 2 alone:\n%q", code, out)
	}

	// Several marks at the very start all go, and one note says how many.
	for n, want := range map[int]string{2: "standard input begins with 2 " + note, 3: "standard input begins with 3 " + note} {
		out, errOut, code := run(strings.Repeat(bom, n)+"one\n", "read", "-")
		if code != 0 || strings.Contains(out, bom) || strings.Count(errOut, want) != 1 || strings.Count(errOut, "\n") != 1 {
			t.Errorf("%d leading marks: exit %d, want one line holding %q:\n%q\n%s", n, code, want, out, errOut)
		}
	}

	// Only LEADING marks go. With a mark at the start and another at the start of
	// line 2, an implementation that strips every mark once it finds a leading one
	// would pass the leading-only and interior-only cases; here it serves 8 bytes
	// and loses line 2's mark (found by a pre-code challenge).
	if out, errOut, code := run(bom+"one\n"+bom+"two\n", "read", "-"); code != 0 || !strings.Contains(out, "==> -  2L  11B") ||
		!strings.Contains(out, " 1| one\n") || !strings.Contains(out, " 2| "+bom+"two\n") || errOut != "" {
		t.Errorf("a leading mark and one on line 2: exit %d, want 2L 11B, line 1 exactly one, line 2 keeping its mark, and no note:\n%q\n%s", code, out, errOut)
	}

	// Controls that must not change.
	// 1. No mark: no note, and line 1 as today.
	if out, errOut, code := run("one\ntwo\n", "read", "-"); code != 0 || errOut != "" || !strings.Contains(out, "==> -  2L  8B") {
		t.Errorf("unmarked input: exit %d:\n%q\n%s", code, out, errOut)
	}
	// 2. A mark that is not at the very start is content and is served.
	if out, errOut, code := run("one\n"+bom+"two\n", "read", "-"); code != 0 || !strings.Contains(out, bom+"two") || errOut != "" {
		t.Errorf("a mark at the start of line 2: exit %d:\n%q\n%s", code, out, errOut)
	}
	// 3. A file keeps its mark: a read of a file serves the bytes it holds.
	if out, _, code := run("", "read", "f.txt"); code != 0 || !strings.Contains(out, bom+"one") {
		t.Errorf("a file's mark was dropped: exit %d:\n%q", code, out)
	}
	// 4. Input that is only a mark is empty input, as it is today without one.
	if out, _, code := run(bom, "read", "-"); code != 0 || !strings.Contains(out, "==> -  0L  0B") {
		t.Errorf("only a mark: exit %d, want an empty stream:\n%q", code, out)
	}
}
