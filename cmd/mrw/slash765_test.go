package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #765. A file named with a trailing separator (b.txt/, b.txt/.) names a
// directory that is not there (ADR-076). A direct read says UNREADABLE of it,
// as Decision 1 decided; every finder now says the same, from the same
// rooted.ErrNotADirectory. A path outside the root, named beside it, is still
// REFUSED, and a plan maker writes nothing.
func TestAFinderSaysUnreadableOfAFileSpelledAsADirectory(t *testing.T) {
	setup := func(t *testing.T) (root, outside string) {
		t.Helper()
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		root = t.TempDir()
		writeFiles(t, root, map[string]string{"a.go": "package a\nvar x = foo(1, 2)\n", "b.txt": "foo\n"})
		return root, "../outside-" + filepath.Base(root)
	}
	unreadable := func(t *testing.T, out, name string) {
		t.Helper()
		if !strings.Contains(out, "==> "+name+"  UNREADABLE  ") || strings.Contains(out, name+"  REFUSED") {
			t.Errorf("%s is not reported UNREADABLE:\n%s", name, out)
		}
	}
	for _, spelled := range []string{"b.txt/", "b.txt/."} {
		t.Run("grep "+spelled, func(t *testing.T) {
			root, outside := setup(t)
			direct, _ := runIn(t, root, "read", spelled)
			unreadable(t, direct, spelled)
			out, code := runIn(t, root, "read", "--grep", "foo", "a.go", spelled, outside)
			if code != 1 || !strings.Contains(out, "==> a.go") {
				t.Errorf("exit %d, want 1 with a.go served:\n%s", code, out)
			}
			unreadable(t, out, spelled)
			if !strings.Contains(out, "==> "+outside+"  REFUSED  ") {
				t.Errorf("the path outside the root is not REFUSED:\n%s", out)
			}
		})
	}
	t.Run("ast_grep", func(t *testing.T) {
		root, _ := setup(t)
		installFakeAstGrep(t, "", 0)
		out, code := runIn(t, root, "read", "--ast-grep", "foo($A, $B)", "b.txt/")
		if code != 1 {
			t.Errorf("exit %d, want 1:\n%s", code, out)
		}
		unreadable(t, out, "b.txt/")
	})
	t.Run("read_grep_sub", func(t *testing.T) {
		root, _ := setup(t)
		var code int
		errOut := stderrOf(t, func() { _, code = runIn(t, root, "read", "--grep", "foo", "--sub", "bar", "a.go", "b.txt/") })
		if code != 1 {
			t.Errorf("exit %d, want 1:\n%s", code, errOut)
		}
		unreadable(t, errOut, "b.txt/")
	})
	t.Run("write_grep_sub", func(t *testing.T) {
		root, _ := setup(t)
		var code int
		errOut := stderrOf(t, func() { _, code = writeIn(t, root, "--grep", "foo", "--sub", "bar", "a.go", "b.txt/") })
		if code != 1 {
			t.Errorf("exit %d, want 1:\n%s", code, errOut)
		}
		unreadable(t, errOut, "b.txt/")
		for _, f := range []string{"a.go", "b.txt"} {
			if b, _ := os.ReadFile(filepath.Join(root, f)); strings.Contains(string(b), "bar") {
				t.Errorf("%s was written beside a path that could not be read:\n%s", f, b)
			}
		}
		// The control: without the bad spelling the same write lands.
		_, code = writeIn(t, root, "--grep", "foo", "--sub", "bar", "a.go")
		if b, _ := os.ReadFile(filepath.Join(root, "a.go")); code != 0 || !strings.Contains(string(b), "bar(1, 2)") {
			t.Errorf("the control did not write a.go: exit %d, %q", code, b)
		}
	})
}
