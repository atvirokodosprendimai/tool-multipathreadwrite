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
