package main

import (
	"strings"
	"testing"
)

// The Codex review of #371, P2. The internal "create" format is reachable only
// through --create; typed as --format=create it is an unknown format, and the
// refusal comes before any attempt to read standard input.
func TestTheInternalCreateFormatIsNotReachableWithoutTheFlag(t *testing.T) {
	root := t.TempDir()
	var out string
	var code int
	withStdin(t, "x\n", func() { out, code = runIn(t, root, "write", "--no-check", "--format", "create") })
	if code != 2 || !strings.Contains(out, "unknown --format") {
		t.Errorf("--format=create without --create: exit %d, want 2 naming an unknown format\n%s", code, out)
	}
}
