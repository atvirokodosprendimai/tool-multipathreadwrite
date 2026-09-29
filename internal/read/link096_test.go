package read

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-096 T1. A path the caller names is never dropped by the walk: a named
// link to a directory, spelled relative to the root, is refused by name with
// the directory to name instead; spelled absolutely it walks the directory it
// resolves to, as v1.31.0 does (M's choice at acceptance); and one that
// resolves to the root is the root.

// linkTree096 holds d/f.go and d/sub/g.go, both matching, and dlink -> d.
func linkTree096(t *testing.T) string {
	t.Helper()
	root := tree(t, map[string]string{"d/f.go": "Target\n", "d/sub/g.go": "Target\n"})
	if err := os.Symlink("d", filepath.Join(root, "dlink")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	return root
}

func TestWalkRefusesANamedLinkToADirectoryAndNamesItsTarget(t *testing.T) {
	root := linkTree096(t)
	if err := os.Symlink("dlink", filepath.Join(root, "dlink2")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	for _, name := range []string{"dlink", "dlink/", "dlink/.", "dlink2"} {
		specs, probs := walk(t, root, "Target", []string{name})
		if len(specs) != 0 {
			t.Errorf("%s: served %v; a named link to a directory is refused, not walked", name, paths(specs))
		}
		if len(probs) != 1 {
			t.Errorf("%s: problems = %v; want exactly one naming the link", name, probs)
			continue
		}
		if probs[0].Path != name {
			t.Errorf("%s: the refusal names %q; it must name the path as the caller wrote it", name, probs[0].Path)
		}
		if !strings.Contains(probs[0].Reason, "is a link to the directory d,") || !strings.HasSuffix(probs[0].Reason, "name d") {
			t.Errorf("%s: reason %q does not name the directory d to name instead", name, probs[0].Reason)
		}
	}
}

func TestWalkStillWalksTheAbsoluteSpellingOfANamedDirectoryLink(t *testing.T) {
	root := linkTree096(t)
	for _, name := range []string{filepath.Join(root, "dlink"), filepath.Join(root, "dlink") + string(filepath.Separator)} {
		specs, probs := walk(t, root, "Target", []string{name})
		if len(probs) != 0 {
			t.Errorf("%s: problems = %v; the absolute spelling walks the directory it resolves to", name, probs)
		}
		if got := strings.Join(paths(specs), ","); got != "d/f.go,d/sub/g.go" {
			t.Errorf("%s: walked %q; want d/f.go,d/sub/g.go under d's own names", name, got)
		}
	}
}

func TestWalkStillWalksTheRootNamedThroughTheLinkItWasGivenBy(t *testing.T) {
	root := tree(t, map[string]string{"top.go": "Target\n", "d/f.go": "Target\n"})
	if err := os.Symlink(".", filepath.Join(root, "self")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	linked := filepath.Join(t.TempDir(), "L")
	if err := os.Symlink(root, linked); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	cases := []struct {
		what, root, name string
	}{
		{"the linked root named absolutely", linked, linked},
		{"relative self -> .", root, "self"},
		{"relative self/ under the linked root", linked, "self/"},
		{". itself", root, "."},
		{". under the linked root", linked, "."},
	}
	for _, c := range cases {
		specs, probs := walk(t, c.root, "Target", []string{c.name})
		if len(probs) != 0 {
			t.Errorf("%s: problems = %v; a path that resolves to the root is the root", c.what, probs)
		}
		if got := strings.Join(paths(specs), ","); got != "d/f.go,top.go" {
			t.Errorf("%s: walked %q; want the root's d/f.go,top.go", c.what, got)
		}
	}
}

func TestWalkStillServesAFileLinkAndAPathThroughALink(t *testing.T) {
	root := linkTree096(t)
	if err := os.Symlink(filepath.Join("d", "f.go"), filepath.Join(root, "flink")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "x"), []byte("Target\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"d":         "d/f.go,d/sub/g.go",
		"dlink/sub": "dlink/sub/g.go",
		"flink":     "flink",
		".git":      ".git/x",
	}
	for name, want := range cases {
		specs, probs := walk(t, root, "Target", []string{name})
		if len(probs) != 0 {
			t.Errorf("%s: problems = %v; v1.31.0 served this shape", name, probs)
		}
		if got := strings.Join(paths(specs), ","); got != want {
			t.Errorf("%s: walked %q, want %q", name, got, want)
		}
	}
}
