package read

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// ADR-135, and the rule ADR-138 set for links: the caller's own rules come
// first. A name Windows will not keep that the caller excluded is not reported,
// and one the ignore rules name is the ignored file it is, not an unkeepable
// name. Driven on Windows only, as the refusal of the name is.
func TestAnExcludedOrIgnoredUnkeepableNameIsNotCountedAsUnkeepable(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("needle\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	p := `\\?\` + filepath.Join(root, "aux.txt")
	if err := os.WriteFile(p, []byte("needle\n"), 0o600); err != nil {
		t.Fatalf("could not make aux.txt: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(p) })

	var sk WalkSkipped
	if _, _, err := Walk(root, nil, WalkOptions{Pattern: regexp.MustCompile("needle"), Exclude: []string{"aux.txt"}, Skipped: &sk}); err != nil {
		t.Fatal(err)
	}
	if sk != (WalkSkipped{}) {
		t.Errorf("an excluded unkeepable name was reported: %+v", sk)
	}

	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("aux.txt\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	sk = WalkSkipped{}
	if _, _, err := Walk(root, nil, WalkOptions{Pattern: regexp.MustCompile("needle"), Skipped: &sk}); err != nil {
		t.Fatal(err)
	}
	if sk.Unkeepable != 0 || sk.Ignored != 1 {
		t.Errorf("an ignored unkeepable name: %+v, want it counted as ignored (1) and not as unkeepable", sk)
	}
}
