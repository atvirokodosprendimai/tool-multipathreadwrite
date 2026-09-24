package guide

import (
	"strings"
	"testing"
)

func TestEverySurfaceContainsTheSharedSentences(t *testing.T) {
	got := Shared()
	for _, must := range []string{
		"Use mrw always: plan the activity as one read of every site, then one plan, then one write.",
		"nothing is written",
		"per line, not per file",
		"models no target syntax",
		"names the file, the plan line, and the reason",
	} {
		if !strings.Contains(got, must) {
			t.Errorf("Shared() does not teach %q:\n%s", must, got)
		}
	}
	if strings.Contains(got, "@@") {
		t.Error("Shared() embeds a plan example; examples stay in the MCP file that executes them")
	}
	if strings.Contains(got, WhyAllOrNothing()) {
		t.Error("Shared() absorbed the why; it stays extra on CLI() and the handshake")
	}
}

func TestCLIContainsSharedAndTheOperatorTraps(t *testing.T) {
	got := CLI()
	if !strings.Contains(got, Shared()) {
		t.Error("CLI() does not contain Shared() verbatim")
	}
	for _, must := range []string{
		WhyAllOrNothing(),
		"through a pipe",
		"Exit 3",
		"MSYS",
		"glob",
		`anchor="`,
		"single-quot",
		"body=",
		"line count",
		"lines=",
		"-C",
		"--root",
		"Python",
	} {
		if !strings.Contains(got, must) {
			t.Errorf("CLI() does not teach %q:\n%s", must, got)
		}
	}
}

func TestCLITeachesAlwaysAndAPlan(t *testing.T) {
	got := CLI()
	for _, must := range []string{
		"Use mrw always",
		"one read of every site, then one plan, then one write",
		"@@ path 0 create",
	} {
		if !strings.Contains(got, must) {
			t.Errorf("CLI() does not teach %q:\n%s", must, got)
		}
	}
	if strings.Contains(got, "3 or more edits") {
		t.Error("CLI still teaches the threshold that trains agents never to use mrw")
	}
	if strings.Contains(Shared(), "@@ path 0 create") {
		t.Error("Shared() absorbed the cookbook; the handshake would carry it")
	}
}

// TestCLITeachesTheReadSide holds ADR-063: a caller with only the binary is
// told to plan one read of every site, so the document must teach the address
// forms and the flags that find sites it cannot name — and where a plan reads
// an address differently from a read.
func TestCLITeachesTheReadSide(t *testing.T) {
	got := CLI()
	for _, must := range []string{
		"'b.go:/func Start/,+12'",
		"PATH:RANGE[,RANGE...]",
		"N- (to the end)",
		"-M (from the start)",
		"A,+N",
		"$ (the last line)",
		"/from/,/to/",
		"but not -M or a comma list",
		"must match exactly once",
		"--grep PATTERN",
		"--exclude GLOB",
		"--ast-grep PATTERN",
		"--files-from FILE",
		"rg -l X . |",
		"A read exits 1 when a range cannot be served",
		"an end past the last line is clamped, not an error",
		"a .git directory the walk meets is skipped, but one you name is walked",
		".gitignore",
	} {
		if !strings.Contains(got, must) {
			t.Errorf("CLI() does not teach %q:\n%s", must, got)
		}
	}
	for _, leak := range []string{"--grep", "--files-from", "exactly once"} {
		if strings.Contains(Shared(), leak) {
			t.Errorf("Shared() absorbed %q; the read section stays on CLI()", leak)
		}
	}
}

// ADR-066 (Codex review of #207). A commit that fails after some files landed
// is reported PARTIALLY APPLIED, so the Shared sentence both surfaces serve
// cannot promise that any failed hunk means nothing was written.
func TestSharedSaysAFailedCommitIsReportedPartial(t *testing.T) {
	got := Shared()
	if strings.Contains(got, "if any hunk fails, nothing is written") {
		t.Errorf("Shared still promises nothing is written whenever a hunk fails:\n%s", got)
	}
	if !strings.Contains(got, "PARTIALLY APPLIED") {
		t.Errorf("Shared does not say a failed commit is reported PARTIALLY APPLIED:\n%s", got)
	}
}
