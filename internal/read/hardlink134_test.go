package read

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// ADR-134. A walk dropped the ledger itself and still matched a hard link to
// it, because the boundary compared the path and never the file.
func TestAWalkDropsAHardLinkToMrwsState(t *testing.T) {
	root, st := t.TempDir(), t.TempDir()
	t.Setenv("XDG_STATE_HOME", st)
	ledger := filepath.Join(st, "mrw", "k", "seen")
	if err := os.MkdirAll(filepath.Dir(ledger), 0o700); err != nil {
		t.Fatal(err)
	}
	for p, body := range map[string]string{ledger: "needle\n", filepath.Join(root, "a.txt"): "needle\n"} {
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Link(ledger, filepath.Join(root, "hl.txt")); err != nil {
		t.Skipf("hard links are not available here: %v", err)
	}
	specs, _, err := Walk(root, nil, WalkOptions{Pattern: regexp.MustCompile("needle")})
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 1 || filepath.ToSlash(specs[0].Path) != "a.txt" {
		t.Errorf("the walk found %v, want a.txt alone", specs)
	}
}
