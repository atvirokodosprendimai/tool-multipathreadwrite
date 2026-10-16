package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-161 Decision 14. The built binary, run in two directories of one
// checkout with no --root: what a read in one licenses, a write from the other
// applies. The pair is a tree with no .git, where the reads stay per directory.
func TestAReadInASubdirectoryLicensesAWriteFromTheTop(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "mrw.exe") // .exe: Windows exec will not launch an extensionless file
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	state := t.TempDir()
	run := func(dir, stdin string, args ...string) (string, string, int) {
		t.Helper()
		c := exec.Command(bin, args...)
		c.Dir = dir
		c.Env = append(os.Environ(), "XDG_STATE_HOME="+state)
		c.Stdin = strings.NewReader(stdin)
		var o, e strings.Builder
		c.Stdout, c.Stderr = &o, &e
		err := c.Run()
		if ee, ok := err.(*exec.ExitError); ok {
			return o.String(), e.String(), ee.ExitCode()
		} else if err != nil {
			t.Fatal(err)
		}
		return o.String(), e.String(), 0
	}
	tree := func(git bool) (top, sub string) {
		top = t.TempDir()
		sub = filepath.Join(top, "sub")
		subTree(t, top, map[string]string{"sub/a.txt": "one\ntwo\n"})
		if git {
			if err := os.Mkdir(filepath.Join(top, ".git"), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		return top, sub
	}

	top, sub := tree(true)
	if _, e, code := run(sub, "", "read", "a.txt"); code != 0 {
		t.Fatalf("read in sub: exit %d: %s", code, e)
	}
	// mrw seen from sub names the checkout's directory first and, because the
	// tally and the working set stay this root's own, that one on a second line.
	so, _, _ := run(sub, "", "seen")
	to, _, _ := run(top, "", "seen")
	sl, tl := strings.Split(so, "\n"), strings.Split(to, "\n")
	if sl[0] != tl[0] || !strings.HasPrefix(sl[1], "# this root's tally and working set: ") || strings.Contains(sl[0], sl[1]) {
		t.Errorf("mrw seen from sub:\n%s\nfrom the top:\n%s", so, to)
	}
	if strings.Contains(to, "this root's tally") {
		t.Errorf("mrw seen at the top names a second directory:\n%s", to)
	}
	if o, e, code := run(top, "@@ sub/a.txt 1 replace\nONE\n", "write", "-"); code != 0 {
		t.Fatalf("a read in sub did not license a write from the top: exit %d\n%s%s", code, o, e)
	}
	if got := subFile(t, top, "sub/a.txt"); got != "ONE\ntwo\n" {
		t.Errorf("sub/a.txt = %q after the write from the top", got)
	}

	top, sub = tree(true)
	if _, e, code := run(top, "", "read", "sub/a.txt"); code != 0 {
		t.Fatalf("read at the top: exit %d: %s", code, e)
	}
	if o, e, code := run(sub, "@@ a.txt 1 replace\nONE\n", "write", "-"); code != 0 {
		t.Fatalf("a read at the top did not license a write from sub: exit %d\n%s%s", code, o, e)
	}
	if got := subFile(t, top, "sub/a.txt"); got != "ONE\ntwo\n" {
		t.Errorf("sub/a.txt = %q after the write from sub", got)
	}

	top, sub = tree(false)
	if _, e, code := run(sub, "", "read", "a.txt"); code != 0 {
		t.Fatalf("read in sub, no .git: exit %d: %s", code, e)
	}
	o, e, code := run(top, "@@ sub/a.txt 1 replace\nONE\n", "write", "-")
	if code != 1 || !strings.Contains(o+e, "has not been read") {
		t.Errorf("with no .git a read in sub licensed a write from the top: exit %d\n%s%s", code, o, e)
	}
	if got := subFile(t, top, "sub/a.txt"); got != "one\ntwo\n" {
		t.Errorf("the refused write changed the file: %q", got)
	}
}
