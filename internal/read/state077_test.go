package read

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// ADR-077. A walk under a root that holds mrw's state searched the ledger and
// the ack store as if they were the caller's files. Every discovered path
// passes rooted.Resolve (ADR-007 rule 3), which refuses them, so the walk
// serves the caller's file and nothing of mrw's.
func TestTheWalkServesNothingFromMrwsState(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, ".st"))
	for name, body := range map[string]string{".st/mrw/k/seen": "needle\n", ".st/mrw/k/pending.json": "needle\n", "a.txt": "needle\n"} {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	specs, _, err := Walk(root, nil, WalkOptions{Pattern: regexp.MustCompile("needle")})
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 1 || filepath.ToSlash(specs[0].Path) != "a.txt" {
		t.Errorf("the walk found %v, want a.txt alone", specs)
	}
}
