package read

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// A refusal that is not about the name — a link out of the root — stays
// uncounted: it is the path whose existence ADR-007 rule 2 keeps shut (ADR-135).
func TestAnEscapeIsNotCountedAsAName(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("needle\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "o.txt")
	if err := os.WriteFile(outside, []byte("needle\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "esc.txt")); err != nil {
		t.Skipf("links are not available here: %v", err)
	}
	var sk WalkSkipped
	if _, _, err := Walk(root, nil, WalkOptions{Pattern: regexp.MustCompile("needle"), Skipped: &sk}); err != nil {
		t.Fatal(err)
	}
	if sk != (WalkSkipped{}) {
		t.Errorf("a link out of the root skipped %+v, want nothing said", sk)
	}
}
