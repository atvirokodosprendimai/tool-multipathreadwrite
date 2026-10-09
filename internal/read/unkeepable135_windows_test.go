package read

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// ADR-135. A walk dropped a discovered file whose name Windows will not keep
// without a word, while it counts ignored files, binaries and nested
// repositories: four Windows sessions got a grep that returned fewer files than
// exist and nothing saying why. The rest is served, the names are counted, and
// the note says so. The names are made through \\?\ paths, since Windows will not
// create them otherwise; only a Windows build refuses them, so this is where the
// walk can be driven.
func TestAWalkCountsTheNamesWindowsWillNotKeep(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("needle\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"aux.txt", "trail."} {
		p := `\\?\` + filepath.Join(root, n)
		if err := os.WriteFile(p, []byte("needle\n"), 0o600); err != nil {
			t.Fatalf("could not make %s: %v", n, err)
		}
		t.Cleanup(func() { _ = os.Remove(p) })
	}
	var sk WalkSkipped
	specs, _, err := Walk(root, nil, WalkOptions{Pattern: regexp.MustCompile("needle"), Skipped: &sk})
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 1 || filepath.ToSlash(specs[0].Path) != "a.txt" {
		t.Errorf("the walk served %v, want a.txt alone", specs)
	}
	if sk.Unkeepable != 2 {
		t.Errorf("Unkeepable = %d, want 2 (aux.txt, trail.)", sk.Unkeepable)
	}
	note := SkipNote(sk, "--no-ignore")
	if !strings.Contains(note, "2 file(s) with a name Windows will not keep") || strings.Contains(note, "--no-ignore walks them") {
		t.Errorf("note = %q, want the count and no promise that --no-ignore walks them", note)
	}
}
