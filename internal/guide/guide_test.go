package guide

import (
	"strings"
	"testing"
)

func TestEverySurfaceContainsTheSharedSentences(t *testing.T) {
	got := Shared()
	for _, must := range []string{
		"3 or more edits, 2 or more files, or several ranges you need to read",
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
}

func TestCLIContainsSharedAndTheOperatorTraps(t *testing.T) {
	got := CLI()
	if !strings.Contains(got, Shared()) {
		t.Error("CLI() does not contain Shared() verbatim")
	}
	for _, must := range []string{
		"through a pipe",
		"Exit 3",
		"MSYS",
		"glob",
	} {
		if !strings.Contains(got, must) {
			t.Errorf("CLI() does not teach %q:\n%s", must, got)
		}
	}
}
