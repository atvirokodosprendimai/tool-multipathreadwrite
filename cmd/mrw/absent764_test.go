package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #764. A named path that is not there is UNREADABLE under every finder, as a
// direct read says of it: --grep, --ast-grep, --count, read --sub and the two
// write plan makers. A dangling link inside the root is the same failure (the
// OS answers ENOENT when it is followed) and gets the direct read's word too.
// A path outside the root is still REFUSED: that is mrw's own rule.
func TestAFinderSaysUnreadableOfANamedPathThatIsNotThere(t *testing.T) {
	setup := func(t *testing.T) (root, outside string) {
		t.Helper()
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		root = t.TempDir()
		writeFiles(t, root, map[string]string{"a.go": "package a\nfunc foo(a, b int) {}\nvar x = foo(1, 2)\n"})
		return root, "../outside-" + filepath.Base(root)
	}
	unreadable := func(t *testing.T, out, name string) {
		t.Helper()
		if !strings.Contains(out, "==> "+name+"  UNREADABLE  ") || strings.Contains(out, name+"  REFUSED") {
			t.Errorf("%s is not reported UNREADABLE:\n%s", name, out)
		}
	}
	refused := func(t *testing.T, out, name string) {
		t.Helper()
		if !strings.Contains(out, "==> "+name+"  REFUSED  ") || strings.Contains(out, name+"  UNREADABLE") {
			t.Errorf("%s is not reported REFUSED:\n%s", name, out)
		}
	}

	t.Run("grep", func(t *testing.T) {
		root, outside := setup(t)
		out, code := runIn(t, root, "read", "--grep", "foo", "a.go", "absent.go", outside)
		if code != 1 || !strings.Contains(out, "==> a.go") {
			t.Errorf("exit %d, want 1 with a.go served:\n%s", code, out)
		}
		unreadable(t, out, "absent.go")
		refused(t, out, outside)
	})
	t.Run("grep_dangling_link_matches_the_direct_read", func(t *testing.T) {
		root, _ := setup(t)
		if err := os.Symlink("nowhere.go", filepath.Join(root, "dangling.go")); err != nil {
			t.Skipf("this OS cannot make a symbolic link here: %v", err)
		}
		direct, _ := runIn(t, root, "read", "dangling.go")
		unreadable(t, direct, "dangling.go")
		out, _ := runIn(t, root, "read", "--grep", "foo", "dangling.go")
		unreadable(t, out, "dangling.go")
	})
	t.Run("count", func(t *testing.T) {
		root, _ := setup(t)
		out, code := runIn(t, root, "read", "--grep", "foo", "--count", "a.go", "absent.go")
		if code != 1 {
			t.Errorf("exit %d, want 1:\n%s", code, out)
		}
		unreadable(t, out, "absent.go")
	})
	t.Run("ast_grep", func(t *testing.T) {
		root, outside := setup(t)
		installFakeAstGrep(t, "", 0)
		out, code := runIn(t, root, "read", "--ast-grep", "foo($A, $B)", "absent.go", outside)
		if code != 1 {
			t.Errorf("exit %d, want 1:\n%s", code, out)
		}
		unreadable(t, out, "absent.go")
		refused(t, out, outside)
	})
	t.Run("read_grep_sub", func(t *testing.T) {
		root, _ := setup(t)
		// A plan maker prints its plan on standard output and its problems on
		// standard error, so both are read.
		var out string
		var code int
		errOut := stderrOf(t, func() { out, code = runIn(t, root, "read", "--grep", "foo", "--sub", "bar", "a.go", "absent.go") })
		if code != 1 {
			t.Errorf("exit %d, want 1:\n%s%s", code, out, errOut)
		}
		unreadable(t, errOut, "absent.go")
	})
	t.Run("write_grep_sub", func(t *testing.T) {
		root, _ := setup(t)
		var code int
		errOut := stderrOf(t, func() { _, code = writeIn(t, root, "--grep", "foo", "--sub", "bar", "a.go", "absent.go") })
		if code != 1 {
			t.Errorf("exit %d, want 1:\n%s", code, errOut)
		}
		unreadable(t, errOut, "absent.go")
		if b, _ := os.ReadFile(filepath.Join(root, "a.go")); strings.Contains(string(b), "bar") {
			t.Errorf("a.go was written beside a path that could not be read:\n%s", b)
		}
	})
	t.Run("write_ast_grep_rewrite", func(t *testing.T) {
		root, _ := setup(t)
		// The fake reports a hit in a.go, so the check that nothing was written
		// can fail: without the absent path the same call rewrites a.go (below).
		body, err := os.ReadFile(filepath.Join(root, "a.go"))
		if err != nil {
			t.Fatal(err)
		}
		installFakeAstGrep(t, rwStream(t, "a.go", string(body), [2]string{"foo(1, 2)", "bar(2, 1)"}), 0)
		var code int
		errOut := stderrOf(t, func() {
			_, code = writeIn(t, root, "--ast-grep", "foo($A, $B)", "--rewrite", "bar($B, $A)", "a.go", "absent.go")
		})
		if code != 1 {
			t.Errorf("exit %d, want 1:\n%s", code, errOut)
		}
		unreadable(t, errOut, "absent.go")
		if b, _ := os.ReadFile(filepath.Join(root, "a.go")); strings.Contains(string(b), "bar") {
			t.Errorf("a.go was written beside a path that could not be read:\n%s", b)
		}
		// The control: a.go alone is rewritten by the same fake.
		errOut = stderrOf(t, func() {
			_, code = writeIn(t, root, "--ast-grep", "foo($A, $B)", "--rewrite", "bar($B, $A)", "a.go")
		})
		if b, _ := os.ReadFile(filepath.Join(root, "a.go")); code != 0 || !strings.Contains(string(b), "bar(2, 1)") {
			t.Errorf("the control did not rewrite a.go: exit %d, a.go %q:\n%s", code, b, errOut)
		}
	})
}
