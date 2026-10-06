//go:build darwin

package apply

import (
	"os"
	"path/filepath"
	"testing"
)

// ADR-086 on a real filesystem. APFS refuses a name that is not valid UTF-8 on
// create and rename (EILSEQ) while lstat of it answers "does not exist", so
// nothing before commit noticed: a rename to moved/\xffdash.txt left the plan
// PARTIALLY APPLIED (chaos seed 25). Skipped on a volume that accepts the byte.
func TestAnAPFSInvalidNameIsRefusedBeforeAnyWrite(t *testing.T) {
	check := t.TempDir()
	if f, err := os.OpenFile(filepath.Join(check, "p\xff"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600); err == nil {
		f.Close()
		t.Skip("this volume accepts a name that is not valid UTF-8; there is nothing to refuse")
	}
	for _, tc := range []struct {
		name string
		last Input
	}{
		{"create", Input{Path: "bad\xffname.txt", Op: "create", Body: []string{"x"}, Lines: -1, Index: 1}},
		{"rename", Input{Path: "b.txt", Op: "rename", Body: []string{"moved/bad\xffname.txt"}, Lines: -1, Index: 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			write(t, root, "a.txt", "a\n")
			write(t, root, "b.txt", "b\n")
			res, err := Apply(root, []Input{
				{Path: "a.txt", Start: 1, End: 1, Op: "replace", Body: []string{"A"}, Lines: -1, Index: 0},
				tc.last,
			}, Options{Force: true})
			// ADR-132: EILSEQ is a name the system refuses — refused, exit 1.
			if err != nil || res.Applied || res.Failed != 1 {
				t.Fatalf("an APFS-invalid name: err=%v applied=%v failed=%d, want no error, not applied, one failed hunk", err, res.Applied, res.Failed)
			}
			if got := read(t, root, "a.txt"); got != "a\n" {
				t.Errorf("a.txt = %q: the content edit landed beside a refused name", got)
			}
			entries, err := os.ReadDir(root)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 2 {
				t.Errorf("the tree holds %d entries, want a.txt and b.txt only", len(entries))
			}
		})
	}
}
