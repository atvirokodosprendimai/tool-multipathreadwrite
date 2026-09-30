package apply

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-086. A name the filesystem cannot hold was found only at commit, after
// the plan's other files had been renamed into place: PARTIALLY APPLIED for a
// rename, and for a create the same once the header keeps the name (T1). The
// probe at staging refuses it first, so the hunk fails and nothing is written.
// Driven through probeNameFn so it runs on every platform; the darwin test
// drives a real APFS refusal.
func TestANameTheFilesystemRefusesWritesNothing(t *testing.T) {
	real := probeNameFn
	t.Cleanup(func() { probeNameFn = real })
	probeNameFn = func(tr *tree, p string) error {
		if strings.HasSuffix(p, "refused.txt") {
			return errors.New("illegal byte sequence")
		}
		return real(tr, p)
	}
	for _, tc := range []struct {
		name string
		last Input
	}{
		{"create", Input{Path: "refused.txt", Op: "create", Body: []string{"x"}, Lines: -1, Index: 1}},
		{"rename", Input{Path: "b.txt", Op: "rename", Body: []string{"moved/refused.txt"}, Lines: -1, Index: 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			write(t, root, "a.txt", "a\n")
			write(t, root, "b.txt", "b\n")
			res, err := Apply(root, []Input{
				{Path: "a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"A"}, Lines: -1, Index: 0},
				tc.last,
			}, Options{Force: true})
			if err == nil || res.Applied || res.Failed != 1 {
				t.Fatalf("a refused name: err=%v applied=%v failed=%d, want an error, not applied, one failed hunk", err, res.Applied, res.Failed)
			}
			if got := read(t, root, "a.txt"); got != "a\n" {
				t.Errorf("a.txt = %q: the plan's content edit landed beside a refused name", got)
			}
			if got := read(t, root, "b.txt"); got != "b\n" {
				t.Errorf("b.txt = %q, want it untouched", got)
			}
			entries, err := os.ReadDir(root)
			if err != nil {
				t.Fatal(err)
			}
			for _, e := range entries {
				if e.Name() != "a.txt" && e.Name() != "b.txt" {
					t.Errorf("a refused plan left %s in the tree", filepath.Join(root, e.Name()))
				}
			}
		})
	}
}
