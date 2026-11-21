package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// #752. A directory a walk cannot list is an I/O failure, not a refusal by policy,
// and a directory named to read says UNREADABLE for the same failure, so a walk
// says UNREADABLE too. It stays a problem (ADR-007 rule 5), so the read exits 1
// "incomplete", and every sibling is still served.
func TestAWalkSaysUnreadableOfADirectoryItCannotList(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("a mode-000 directory is still listable here")
	}
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "locked"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFiles(t, root, map[string]string{"a.txt": "needle\n"})
	if err := os.WriteFile(filepath.Join(root, "locked", "b.txt"), []byte("needle\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(root, "locked"), 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(root, "locked"), 0o755) })
	if err := os.WriteFile(filepath.Join(root, "denied.txt"), []byte("needle\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(root, "denied.txt"), 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(root, "denied.txt"), 0o644) })
	out, code := runIn(t, root, "read", "--grep", "needle", ".")
	if code != 1 {
		t.Errorf("exit %d, want 1: a directory that could not be listed leaves the answer incomplete", code)
	}
	if !strings.Contains(out, "==> locked  UNREADABLE  ") || strings.Contains(out, "locked  REFUSED") {
		t.Errorf("the unlistable directory is not named UNREADABLE:\n%s", out)
	}
	// A FILE the walk cannot open is the same failure, so the same word.
	if !strings.Contains(out, "==> denied.txt  UNREADABLE  ") || strings.Contains(out, "denied.txt  REFUSED") {
		t.Errorf("the unopenable file is not named UNREADABLE:\n%s", out)
	}
	if !strings.Contains(out, "==> a.txt") || !strings.Contains(out, "incomplete:") {
		t.Errorf("the sibling is not served, or the exit does not say incomplete:\n%s", out)
	}
	// The pair: a refusal by policy keeps its word.
	out, _ = runIn(t, root, "read", "--grep", "needle", "../outside-"+filepath.Base(root))
	if !strings.Contains(out, "  REFUSED  ") || strings.Contains(out, "  UNREADABLE  ") {
		t.Errorf("a path outside the root is no longer REFUSED:\n%s", out)
	}
}

// lockedTree is a root with a readable file and a mode-000 directory.
func lockedTree(t *testing.T) string {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "locked"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFiles(t, root, map[string]string{"a.txt": "needle\n"})
	if err := os.WriteFile(filepath.Join(root, "locked", "b.txt"), []byte("needle\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(root, "locked"), 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(root, "locked"), 0o755) })
	return root
}

// #762 review. A directory NAMED as the walk's start that cannot be listed is the
// same failure, so the same word; and a plan maker (write --grep --sub, and the
// ast-grep rewrite) says UNREADABLE of what its walk could not list, as read does.
func TestAWalkStartAndAPlanMakerSayUnreadableOfADirectoryTheyCannotList(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("a mode-000 directory is still listable here")
	}
	root := lockedTree(t)
	out, code := runIn(t, root, "read", "--grep", "needle", "locked")
	if code != 1 || !strings.Contains(out, "==> locked  UNREADABLE  ") || strings.Contains(out, "locked  REFUSED") {
		t.Errorf("a named start that cannot be listed: exit %d:\n%s", code, out)
	}
	// A plan maker's walk reports on standard error, beside the plan it builds.
	errOut := stderrOf(t, func() {
		_, _ = writeIn(t, root, "--dry-run", "--grep", "needle", "--sub", "thread", ".")
	})
	if !strings.Contains(errOut, "==> locked  UNREADABLE  ") || strings.Contains(errOut, "locked  REFUSED") {
		t.Errorf("write --grep --sub names the directory REFUSED, or not at all:\n%s", errOut)
	}
	installFakeAstGrep(t, "", 0)
	errOut = stderrOf(t, func() {
		_, _ = writeIn(t, root, "--dry-run", "--ast-grep", "foo($A)", "--rewrite", "bar($A)", ".")
	})
	if !strings.Contains(errOut, "==> locked  UNREADABLE  ") || strings.Contains(errOut, "locked  REFUSED") {
		t.Errorf("write --ast-grep --rewrite names the directory REFUSED, or not at all:\n%s", errOut)
	}
}

// #762 review. The same classification on the ast-grep path: a mode-000 directory
// NAMED to an ast-grep search is UNREADABLE, as under --grep, not REFUSED.
func TestAnAstGrepSearchSaysUnreadableOfANamedDirectoryItCannotList(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("a mode-000 directory is still listable here")
	}
	root := lockedTree(t)
	installFakeAstGrep(t, "", 0)
	out, _ := runIn(t, root, "read", "--ast-grep", "foo($A)", "locked")
	if !strings.Contains(out, "==> locked  UNREADABLE  ") || strings.Contains(out, "locked  REFUSED") {
		t.Errorf("an ast-grep search names the unlistable directory REFUSED, or not at all:\n%s", out)
	}
}

// #762 final pass. An implicit start (no path named) that cannot be listed is the
// same failure as a named one, under --ast-grep as under --grep: UNREADABLE and
// incomplete, never a false "no match"; a rewrite with no path applies nothing and
// says so.
func TestAnAstGrepSearchWithNoPathSaysUnreadableOfARootItCannotList(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("a mode-000 directory is still listable here")
	}
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	writeFiles(t, root, map[string]string{"a.go": "package main\n"})
	if err := os.Chmod(root, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(root, 0o755) })
	installFakeAstGrep(t, "", 0)
	var out string
	var code int
	errOut := stderrOf(t, func() { out, code = runIn(t, root, "read", "--ast-grep", "package main") })
	all := out + errOut
	if code != 1 || !strings.Contains(all, "  UNREADABLE  ") || !strings.Contains(all, "incomplete:") || strings.Contains(all, "no match:") {
		t.Errorf("exit %d; want UNREADABLE and incomplete, not no match:\n%s", code, all)
	}
	errOut = stderrOf(t, func() {
		out, code = writeIn(t, root, "--dry-run", "--ast-grep", "package main", "--rewrite", "package other")
	})
	all = out + errOut
	if code == 0 || !strings.Contains(all, "  UNREADABLE  ") {
		t.Errorf("a rewrite with no path under an unlistable root: exit %d; want it to apply nothing and say UNREADABLE:\n%s", code, all)
	}
}

// #762 final pass. The walk's own stat of a NAMED path that fails with the OS (a
// permission) is UNREADABLE under both finders, as a direct read says; a refusal by
// the boundary (outside the root, a link that leaves it) stays REFUSED.
func TestANamedPathWhoseStatFailsIsUnreadableAndABoundaryRefusalIsNot(t *testing.T) {
	if runtime.GOOS == "windows" || os.Getuid() == 0 {
		t.Skip("a mode-000 directory is still listable here")
	}
	root := lockedTree(t)
	installFakeAstGrep(t, "", 0)
	for _, finder := range [][]string{{"--grep", "needle"}, {"--ast-grep", "foo($A)"}} {
		args := append(append([]string{"read"}, finder...), "locked/b.txt")
		out, code := runIn(t, root, args...)
		if code != 1 || !strings.Contains(out, "==> locked/b.txt  UNREADABLE  ") || strings.Contains(out, "locked/b.txt  REFUSED") {
			t.Errorf("%v: exit %d; want locked/b.txt UNREADABLE:\n%s", finder, code, out)
		}
	}
	if err := os.Symlink(os.TempDir(), filepath.Join(root, "out")); err != nil {
		t.Skipf("no symlinks here: %v", err)
	}
	for _, finder := range [][]string{{"--grep", "needle"}, {"--ast-grep", "foo($A)"}} {
		for _, named := range []string{"../outside-" + filepath.Base(root), "out/x.txt"} {
			args := append(append([]string{"read"}, finder...), named)
			out, _ := runIn(t, root, args...)
			if !strings.Contains(out, "  REFUSED  ") || strings.Contains(out, "  UNREADABLE  ") {
				t.Errorf("%v %s: a boundary refusal is not REFUSED:\n%s", finder, named, out)
			}
		}
	}
}
