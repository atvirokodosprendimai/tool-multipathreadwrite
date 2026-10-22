package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Issue #496. A subagent told to read a file outside the root was refused with
// "read it with --root pointed where you mean" and fell back to sed: the
// refusal named the flag, not the command. It now prints the command, for an
// absolute path and for ../ alike. The pairs: a path that is not outside, and
// one in mrw's own state directory (no --root lifts that refusal, ADR-157),
// carry no such command.
func TestAnOutOfRootReadNamesTheCommandThatReadsIt(t *testing.T) {
	parent, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root, other := filepath.Join(parent, "root"), filepath.Join(parent, "other")
	for _, d := range []string{root, other} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for path, body := range map[string]string{filepath.Join(other, "hosts"): "one\ntwo\n", filepath.Join(root, "in.txt"): "in\n"} {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	command := "mrw --root " + other + " read hosts:1"
	for name, spec := range map[string]string{
		"an absolute path": filepath.Join(other, "hosts") + ":1",
		"a path with ../":  filepath.Join("..", "other", "hosts") + ":1",
	} {
		out, code := runIn(t, root, "read", spec)
		if code != 1 || !strings.Contains(out, "pointed where you mean: "+command) {
			t.Errorf("%s: exit %d; want 1 naming %q:\n%s", name, code, command, out)
		}
	}
	if out, code := runIn(t, root, "read", "in.txt"); code != 0 || strings.Contains(out, "mrw --root") {
		t.Errorf("a file inside the root: exit %d:\n%s", code, out)
	}
}
