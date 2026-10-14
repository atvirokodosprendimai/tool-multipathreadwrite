package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The Codex review of #373. Two hints are true only with a limit, and say so:
// Go's \b is ASCII-only, and a literal holding \E cannot be wrapped in \Q…\E.
// The limits are run here, so the sentences cannot claim more than the binary does.
func TestTheWordAndLiteralHintsNameTheirLimits(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "u.txt"), []byte("café x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// \b…\b misses a standalone non-ASCII word: the -w sentence must say ASCII.
	if _, code := runIn(t, root, "read", "--grep", `\bcaf\x{e9}\b`, "u.txt"); code != 1 {
		t.Errorf(`\b around a non-ASCII word matched (exit %d): the -w hint's limit is not real`, code)
	}
	if !strings.Contains(foreignFlags["w"], "ASCII") {
		t.Errorf("the -w hint does not name the ASCII limit: %q", foreignFlags["w"])
	}
	// A literal holding \E breaks \Q…\E: the -F sentence must say so.
	if _, code := runIn(t, root, "read", "--grep", `\Qa\Eb\E`, "u.txt"); code == 0 {
		t.Error(`\Q…\E around a literal holding \E was accepted: the -F hint's limit is not real`)
	}
	if !strings.Contains(foreignFlags["F"], `\E`) || !strings.Contains(foreignFlags["F"], "escape") {
		t.Errorf("the -F hint does not name the \\E limit: %q", foreignFlags["F"])
	}
}
