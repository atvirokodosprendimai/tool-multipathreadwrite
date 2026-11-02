package read

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ADR-015: a refusal names the fix. A directory named to read is UNREADABLE, as
// before, and the line after it says a directory is walked and names --grep, in
// prose: no command line is printed, since none runs unchanged in every shell. A
// link names the directory it leads to. A missing file, or one that exists and
// cannot be read, gets no such line.
func TestADirectoryNamedToReadSaysHowToWalkIt(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "sub/a.txt", "a\n")
	const walk = "a directory is walked, not read: give it to mrw read --grep PATTERN"
	cases := []struct{ spec, hint string }{{"sub", walk}, {"sub:1-2", walk}, {"nope.txt", ""}}
	if runtime.GOOS != "windows" {
		if err := os.Symlink("sub", filepath.Join(root, "link")); err != nil {
			t.Fatal(err)
		}
		cases = append(cases, struct{ spec, hint string }{"link", "a link to the directory sub, and a walk does not follow a link: give sub to mrw read --grep"})
		if os.Geteuid() != 0 {
			writeFile(t, root, "locked.txt", "x\n")
			if err := os.Chmod(filepath.Join(root, "locked.txt"), 0); err != nil {
				t.Fatal(err)
			}
			cases = append(cases, struct{ spec, hint string }{"locked.txt", ""})
		}
	}
	for _, c := range cases {
		out, problems := run(t, root, Options{Numbers: true}, c.spec)
		if problems != 1 || !(strings.Contains(out, "UNREADABLE") || strings.Contains(out, "REFUSED")) {
			t.Fatalf("%s: problems=%d, want a refusal:\n%s", c.spec, problems, out)
		}
		walked := strings.Contains(out, "walked, not read") || strings.Contains(out, "does not follow a link")
		if c.hint == "" && walked || c.hint != "" && !strings.Contains(out, c.hint) {
			t.Errorf("%s: want hint %q:\n%s", c.spec, c.hint, out)
		}
	}
}

// The directory hint leaves the shell-split hint as it was: a directory name still
// counts toward it beside an English-word miss (the Codex review of #591).
func TestADirectoryStillCountsTowardTheShellSplitHint(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "internal"), 0o755); err != nil {
		t.Fatal(err)
	}
	out, _ := run(t, root, Options{Numbers: true}, "internal", "missingword")
	if !strings.Contains(out, "walked, not read") || !strings.Contains(out, "shell-split regex address") {
		t.Errorf("want both the walk hint and the shell-split hint:\n%s", out)
	}
}

// A directory's UNREADABLE line says "is a directory" on every OS, never the
// platform's text for reading a directory handle: on Windows that was
// ERROR_INVALID_FUNCTION, "Incorrect function.", which tells a reader nothing
// (the v1.82.0 Windows field test). The hint line under it is unchanged.
func TestADirectoryNamedToReadSaysIsADirectoryOnEveryOS(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	out, problems := run(t, root, Options{Numbers: true}, "sub")
	var line string
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, "==> sub  ") {
			line = l
		}
	}
	if problems != 1 || line != "==> sub  UNREADABLE  is a directory" || strings.Contains(out, "Incorrect function") {
		t.Errorf("problems=%d, want the line %q:\n%s", problems, "==> sub  UNREADABLE  is a directory", out)
	}
	if !strings.Contains(out, "a directory is walked, not read") {
		t.Errorf("the walk hint is gone:\n%s", out)
	}
}

// Only a directory the read could open is "is a directory" with the walk hint.
// A directory that cannot be opened keeps its own error and gets no hint, since
// the walk the hint names would fail the same way (the go-reviewer pass on #595).
// A Windows directory denied by its ACL is the same case; a mode-000 directory
// is the one a test can build.
func TestADirectoryThatCannotBeOpenedKeepsItsOwnErrorAndNoWalkHint(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a mode-000 directory is a unix permission")
	}
	if os.Geteuid() == 0 {
		t.Skip("root opens a mode-000 directory")
	}
	root := t.TempDir()
	locked := filepath.Join(root, "locked")
	if err := os.Mkdir(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	out, problems := run(t, root, Options{Numbers: true}, "locked")
	if problems != 1 || !strings.Contains(out, "==> locked  UNREADABLE  ") || !strings.Contains(out, "permission denied") {
		t.Errorf("problems=%d, want UNREADABLE with permission denied:\n%s", problems, out)
	}
	if strings.Contains(out, "is a directory") || strings.Contains(out, "walked, not read") {
		t.Errorf("a directory that cannot be opened is reported as a walkable directory:\n%s", out)
	}
}
