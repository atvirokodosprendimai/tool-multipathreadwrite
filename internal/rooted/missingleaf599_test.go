package rooted

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Issue #599 (ADR-007: mrw finds the files it serves and says which path it
// looked for). A root reached through a linked ancestor — /var -> /private/var
// on macOS, a junction on Windows — compares as its target, so an absolute path
// to a MISSING file under it must resolve there too. EvalSymlinks fails on a
// missing leaf, Real returned the path as spelled, and read refused a file
// inside the root as "outside the root" instead of reporting it missing.
// mklink is the platform's link: a symlink on unix, a junction on Windows.
// The same holds for any leaf that does not resolve while its parent does: a
// dangling symlink, or a name under a regular file (the go-reviewer pass of #601).
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
	type tc struct {
		name, path string
		inside     bool
		want       string
	}
	var more []tc
	if runtime.GOOS != "windows" {
		// A dangling symlink: Lstat finds it, EvalSymlinks cannot resolve it.
		// (A junction to nowhere is followed to its target's spelling on
		// Windows, by the relative spelling too, so it is not this case.)
		mklink(t, filepath.Join(base, "real", "T", "dang"), filepath.Join(base, "nowhere"))
		more = append(more, tc{"a dangling symlink leaf", filepath.Join(base, "ln", "T", "dang"), true, ""})
	}
	root := filepath.Join(base, "ln", "T")
	absRoot, err := Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	cases := []tc{
		{"a missing leaf", filepath.Join(root, "missing.py"), true, filepath.Join(absRoot, "missing.py")},
		{"a missing leaf two levels down", filepath.Join(root, "no", "such.py"), true, filepath.Join(absRoot, "no", "such.py")},
		{"an existing file", filepath.Join(root, "here.txt"), true, filepath.Join(absRoot, "here.txt")},
		{"a name under a regular file", filepath.Join(root, "here.txt", "x"), true, filepath.Join(absRoot, "here.txt", "x")},
		// The pair: a missing leaf under a link that leaves the root still
		// resolves outside it, so containment still refuses it.
		{"a missing leaf under a link out of the root", filepath.Join(root, "out", "missing.py"), false, ""},
	}
	for _, c := range append(cases, more...) {
		if c.want == "" && c.inside {
			c.want = filepath.Join(absRoot, filepath.Base(c.path))
		}
		got := Real(c.path)
		if Contains(absRoot, got) != c.inside {
			t.Errorf("%s: Real(%s) = %s; inside the root %s = %v, want %v", c.name, c.path, got, absRoot, !c.inside, c.inside)
		}
		if c.want != "" && got != c.want {
			t.Errorf("%s: Real(%s) = %s, want %s", c.name, c.path, got, c.want)
		}
	}
}
