package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ADR-096 T3. `iter add` filed every stat error as missing and answered "no
// such file … (quote a spec containing spaces)" for a link loop and for a
// denied directory: a hint about word splitting, given for neither. A path
// that is not there keeps that sentence; any other error is the OS's own.

func TestIterAddNamesALinkLoopAsALoop(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := grepTree(t, map[string]string{"a.go": "package a\n"})
	if err := os.Symlink("loop", filepath.Join(root, "loop")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	why := "too many levels of symbolic links"
	if runtime.GOOS == "windows" {
		// rooted.Resolve follows links itself there and refuses the loop
		// before any Stat (internal/rooted/links.go).
		why = "leads through more than 255 links"
	}
	unchanged := func(what string) {
		t.Helper()
		if out, code := runIn(t, root, "iter"); code != 0 || !strings.Contains(out, "0 entr(ies)") {
			t.Errorf("%s: the working set changed (exit %d):\n%s", what, code, out)
		}
	}
	for _, args := range [][]string{{"loop"}, {"a.go", "loop"}} {
		out, code := runIn(t, root, append([]string{"iter", "add"}, args...)...)
		if code != exitUsage {
			t.Errorf("iter add %v exited %d, want %d:\n%s", args, code, exitUsage, out)
		}
		if strings.Contains(out, "no such file") || !strings.Contains(out, why) {
			t.Errorf("iter add %v: want %q and not \"no such file\":\n%s", args, why, out)
		}
		unchanged("iter add " + strings.Join(args, " "))
	}
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		return // mode 000 is not enforced here
	}
	noperm := filepath.Join(root, "noperm")
	if err := os.MkdirAll(noperm, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(noperm, "f"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(noperm, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(noperm, 0o755) })
	out, code := runIn(t, root, "iter", "add", "noperm/f")
	if code != exitUsage || strings.Contains(out, "no such file") || !strings.Contains(out, "permission denied") {
		t.Errorf("iter add noperm/f: exit %d, want %d naming permission denied:\n%s", code, exitUsage, out)
	}
	unchanged("iter add noperm/f")
}

func TestIterAddStillSaysNoSuchFileForAPathThatIsNotThere(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := grepTree(t, map[string]string{"a.go": "package a\n"})
	out, code := runIn(t, root, "iter", "add", "nosuch")
	if code != exitUsage || !strings.Contains(out, "no such file: nosuch (quote a spec containing spaces)") {
		t.Errorf("iter add nosuch: exit %d, want %d with the missing-file sentence:\n%s", code, exitUsage, out)
	}
	if out, code := runIn(t, root, "iter", "add", "a.go"); code != 0 || !strings.Contains(out, "@1   a.go") {
		t.Errorf("iter add a.go: exit %d, want 0 listing a.go:\n%s", code, out)
	}
}
