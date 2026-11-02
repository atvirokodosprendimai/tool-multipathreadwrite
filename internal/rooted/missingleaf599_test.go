package rooted

import (
	"os"
	"path/filepath"
	"testing"
)

// Issue #599 (ADR-007: mrw finds the files it serves and says which path it
// looked for). A root reached through a linked ancestor — /var -> /private/var
// on macOS, a junction on Windows — compares as its target, so an absolute path
// to a MISSING file under it must resolve there too. EvalSymlinks fails on a
// missing leaf, Real returned the path as spelled, and read refused a file
// inside the root as "outside the root" instead of reporting it missing.
// mklink is the platform's link: a symlink on unix, a junction on Windows.
func testAMissingLeafUnderALinkedAncestorResolvesInsideTheRoot(t *testing.T, mklink func(t *testing.T, link, target string)) {
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "real", "T"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "real", "T", "here.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(base, "elsewhere"), 0o755); err != nil {
		t.Fatal(err)
	}
	mklink(t, filepath.Join(base, "ln"), filepath.Join(base, "real"))
	mklink(t, filepath.Join(base, "real", "T", "out"), filepath.Join(base, "elsewhere"))
	root := filepath.Join(base, "ln", "T")
	absRoot, err := Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name, path string
		inside     bool
		want       string
	}{
		{"a missing leaf", filepath.Join(root, "missing.py"), true, filepath.Join(absRoot, "missing.py")},
		{"a missing leaf two levels down", filepath.Join(root, "no", "such.py"), true, filepath.Join(absRoot, "no", "such.py")},
		{"an existing file", filepath.Join(root, "here.txt"), true, filepath.Join(absRoot, "here.txt")},
		// The pair: a missing leaf under a link that leaves the root still
		// resolves outside it, so containment still refuses it.
		{"a missing leaf under a link out of the root", filepath.Join(root, "out", "missing.py"), false, ""},
	} {
		got := Real(c.path)
		if Contains(absRoot, got) != c.inside {
			t.Errorf("%s: Real(%s) = %s; inside the root %s = %v, want %v", c.name, c.path, got, absRoot, !c.inside, c.inside)
		}
		if c.want != "" && got != c.want {
			t.Errorf("%s: Real(%s) = %s, want %s", c.name, c.path, got, c.want)
		}
	}
}
