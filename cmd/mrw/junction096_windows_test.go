//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-096 review. read.Walk built its root with filepath.EvalSymlinks, which
// since Go 1.23 leaves a junction as written: WalkDir Lstat-ed the junction,
// saw no directory (ModeIrregular), and `--grep` naming no path or "." served
// nothing, while a named directory below it was served under a name relative
// to the junction. The root is built with rooted.Real now, which follows a
// junction (ADR-071). The headers are asserted exactly, so a mangled name
// fails as surely as silence does.
func TestGrepUnderAJunctionedRootWalksFromTheRoot(t *testing.T) {
	base := t.TempDir()
	root, alias := filepath.Join(base, "root"), filepath.Join(base, "alias")
	for p, b := range map[string]string{
		filepath.Join(root, "a.txt"):        "inside\n",
		filepath.Join(root, "sub", "x.txt"): "inside\n",
	} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(b), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", alias, root).CombinedOutput(); err != nil {
		t.Fatalf("mklink /J: %v\n%s", err, out)
	}
	cases := []struct {
		what string
		argv []string
		want []string
	}{
		{"no path", []string{"read", "--grep", "inside"}, []string{"==> a.txt  ", "==> sub/x.txt  "}},
		{".", []string{"read", "--grep", "inside", "."}, []string{"==> a.txt  ", "==> sub/x.txt  "}},
		{"sub", []string{"read", "--grep", "inside", "sub"}, []string{"==> sub/x.txt  "}},
	}
	for _, c := range cases {
		out, code := runIn(t, alias, c.argv...)
		if code != 0 {
			t.Errorf("%s under a junctioned root: exit %d:\n%s", c.what, code, out)
			continue
		}
		for _, w := range c.want {
			if !strings.Contains(out, w) {
				t.Errorf("%s under a junctioned root: no header %q:\n%s", c.what, strings.TrimSpace(w), out)
			}
		}
		if strings.Contains(out, "==> ..") {
			t.Errorf("%s under a junctioned root: a path is named relative to the junction:\n%s", c.what, out)
		}
	}
}
