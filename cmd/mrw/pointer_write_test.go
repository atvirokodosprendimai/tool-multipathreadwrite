package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A hunk's path may be a working-set pointer, and it must name exactly one
// file (main.go, the write action). contract.sh drove this end to end; this is
// the Go test BACKLOG asked for, against the BUILT binary, because the pointer
// is resolved in the command wiring that an engine test never reaches.
func TestAPointerHunkPathNamesExactlyOneFile(t *testing.T) {
	// mrw.exe on every platform: Windows exec will not launch an extensionless
	// file even by absolute path (Codex review of #249).
	bin := filepath.Join(t.TempDir(), "mrw.exe")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	state, root := t.TempDir(), t.TempDir()
	for n, b := range map[string]string{"a.txt": "one\n", "b.txt": "two\n"} {
		if err := os.WriteFile(filepath.Join(root, n), []byte(b), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run := func(args ...string) (string, int) {
		c := exec.Command(bin, append([]string{"-C", root}, args...)...)
		c.Env = append(os.Environ(), "XDG_STATE_HOME="+state)
		out, err := c.CombinedOutput()
		if ee, ok := err.(*exec.ExitError); ok {
			return string(out), ee.ExitCode()
		} else if err != nil {
			t.Fatal(err)
		}
		return string(out), 0
	}
	for _, args := range [][]string{{"read", "a.txt", "b.txt"}, {"iter", "add", "a.txt", "b.txt"}} {
		if out, code := run(args...); code != 0 {
			t.Fatalf("%v: exit %d\n%s", args, code, out)
		}
	}
	file := func(n string) string {
		b, err := os.ReadFile(filepath.Join(root, n))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}

	// A pointer that names two entries is refused before anything is written.
	out, code := run("write", planFile(t, "@@ @1-2 1 replace\nX\n"))
	if code != 2 || !strings.Contains(out, "@1-2 names 2 entries; a hunk needs exactly one") {
		t.Errorf("@1-2 as a hunk path: exit %d, want 2 naming the count:\n%s", code, out)
	}
	if file("a.txt") != "one\n" || file("b.txt") != "two\n" {
		t.Errorf("a refused pointer plan changed the tree: a=%q b=%q", file("a.txt"), file("b.txt"))
	}

	// The valid sibling: @2 resolves to b.txt and the write lands there only.
	if out, code := run("write", planFile(t, "@@ @2 1 replace\nTWO\n")); code != 0 {
		t.Fatalf("@2 as a hunk path: exit %d, want 0:\n%s", code, out)
	}
	if file("b.txt") != "TWO\n" || file("a.txt") != "one\n" {
		t.Errorf("@2 did not land on b.txt alone: a=%q b=%q", file("a.txt"), file("b.txt"))
	}
}
