package main

import (
	"strings"
	"testing"
)

// ADR-092 T6 (review of #269). --help prints a flag's value as its default,
// so a --then or --then-sh holding control bytes must reach it quoted, as the
// receipt shows it, not as bytes a terminal acts on.
func TestHelpDoesNotEchoARawStepValue(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	for _, cmd := range []string{"check", "write"} {
		out, _ := runIn(t, root, cmd, "--then", "x\x1b[2Jy", "--then-sh", "true \x1b[31m", "--help")
		if !strings.Contains(out, "--then") {
			t.Fatalf("%s --help printed no --then:\n%s", cmd, out)
		}
		if strings.Contains(out, "\x1b") {
			t.Errorf("%s --help echoes a raw control byte:\n%q", cmd, out)
		}
	}
}
