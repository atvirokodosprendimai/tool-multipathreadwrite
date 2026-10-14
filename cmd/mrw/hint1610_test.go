package main

import (
	"runtime"
	"strings"
	"testing"
)

// The v1.60.0 and v1.61.0 Windows retests: `--create -- 'trail '` was told to put
// `--` before the path, which drops --create. --create takes its path as its own
// argument, kept as given.
func TestTheEdgeWhitespaceHintForCreateKeepsCreate(t *testing.T) {
	root := t.TempDir()
	var out string
	var code int
	note := stderrOf(t, func() {
		withStdin(t, "x\n", func() { out, code = runIn(t, root, "write", "--no-check", "--create", "--", "trail ") })
	})
	got := out + note
	if code != 2 || !strings.Contains(got, "--create 'trail '") || strings.Contains(got, "mrw write -- ") {
		t.Errorf("exit %d, output %q, want a usage error that keeps --create in the advice", code, got)
	}
	// cmd.exe keeps single quotes as characters, so its form is named too (ADR-078).
	if runtime.GOOS == "windows" && !strings.Contains(got, `in cmd.exe: mrw write --create "trail "`) {
		t.Errorf("output %q lacks the cmd.exe form", got)
	}
}
