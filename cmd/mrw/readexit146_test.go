package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-146. A read that exits 1 begins its last line with the kind: `no match:`
// when the search found no file and nothing it was asked for was refused,
// `incomplete:` when part of what was asked for was not served. A search that
// matched no file while a path beside it was refused used to end a plain
// "no file matched" and hid the refusal.
func TestAReadThatExitsOneSaysWhichKind(t *testing.T) {
	root := grepTree(t, map[string]string{"d/f.go": "package d\nWANTED\n", "a.txt": "one\n"})
	linked := os.Symlink("d", filepath.Join(root, "dlink")) == nil

	_, err := readIn(t, root, "--grep", "ABSENT", "d")
	if got := errString(err); !strings.HasPrefix(got, "no match: no file matched /ABSENT/") {
		t.Errorf("a grep that matched nothing: %q, want it to begin `no match: no file matched /ABSENT/`", got)
	}

	_, err = readIn(t, root, "missing.txt")
	if got := errString(err); !strings.HasPrefix(got, "incomplete: 1 range(s) could not be served") {
		t.Errorf("a path that does not exist: %q, want it to begin `incomplete: 1 range(s) could not be served`", got)
	}

	_, err = readIn(t, root, "a.txt", "missing.txt")
	if got := errString(err); !strings.HasPrefix(got, "incomplete: ") {
		t.Errorf("one file served, one missing: %q, want it to begin `incomplete: `", got)
	}

	if linked {
		_, err = readIn(t, root, "--grep", "ABSENT", "d", "dlink")
		got := errString(err)
		if !strings.HasPrefix(got, "incomplete: no file matched /ABSENT/, and 1 path(s) were refused") || strings.HasPrefix(got, "no match:") {
			t.Errorf("a grep that matched nothing beside a refused link: %q, want `incomplete: no file matched /ABSENT/, and 1 path(s) were refused`", got)
		}
	}

	// The kind is on the LAST line a caller reads: a pattern's own line breaks
	// are spelled, and an ADR-140 hint ends the line beside a refusal too.
	_, err = readIn(t, root, "--grep", "ABSENT\nSECOND", "d")
	if got := errString(err); strings.Contains(got, "\n") || !strings.HasPrefix(got, `no match: no file matched /ABSENT\nSECOND/`) {
		t.Errorf("a multi-line pattern: %q, want one line beginning `no match: no file matched /ABSENT\\nSECOND/`", got)
	}
	_, err = readIn(t, root, "--grep", "-i", "missing.txt", "a.txt")
	if got := errString(err); !strings.HasPrefix(got, "incomplete: ") || !strings.HasSuffix(got, "(mrw took the value after --grep as the pattern)") {
		t.Errorf("a hinted pattern beside a refusal: %q, want `incomplete: …` ending with the ADR-140 hint", got)
	}

	if out, err := readIn(t, root, "a.txt"); err != nil {
		t.Errorf("a read that served everything failed: %v\n%s", err, out)
	}
}
