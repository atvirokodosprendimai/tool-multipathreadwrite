package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-187. A write runs no check unless it is asked for: in a tree whose
// declared check fails, a .go write with no flag, or with --no-check, lands and
// exits 0 with no check line; --check runs the check and exits 3. ADR-054's
// default ran it on every non-prose write, and a project whose check is its
// whole gate paid minutes per write (memoryd #102; owner, 2026-11-01).
func TestAWriteRunsNoCheckUnlessAskedFor(t *testing.T) {
	needShell(t)
	for _, tc := range []struct {
		name string
		args []string
		want int
		ran  bool
	}{
		{"no flag", nil, 0, false},
		{"--no-check", []string{"--no-check"}, 0, false},
		{"--check", []string{"--check"}, exitCheckFailed, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("XDG_STATE_HOME", t.TempDir())
			root := checkTree(t)
			if _, err := readIn(t, root, "a.go"); err != nil {
				t.Fatal(err)
			}
			out, code := writeIn(t, root, append(tc.args, planFile(t, goPlan))...)
			if code != tc.want {
				t.Fatalf("exit %d, want %d:\n%s", code, tc.want, out)
			}
			if b, _ := os.ReadFile(filepath.Join(root, "a.go")); !strings.Contains(string(b), "_ = 1") {
				t.Errorf("the write did not land:\n%s", b)
			}
			if ran := strings.Contains(out, "check FAIL"); ran != tc.ran {
				t.Errorf("check ran %v, want %v:\n%s", ran, tc.ran, out)
			}
		})
	}
}
