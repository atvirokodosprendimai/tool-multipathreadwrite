package read

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const lostSeparator = "a path separator may have been lost, as a shell eats an unquoted backslash: did you mean "

// realTempDir is t.TempDir() as the root judge resolves it: on macOS /var is a
// link to /private/var, and an absolute path spelled under /var is refused as
// outside the root (as realRoot096 does for the same reason).
func realTempDir(t *testing.T) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// linkDir makes link lead to the directory target: a symlink, or on Windows a
// junction, which needs no privilege.
func linkDir(t *testing.T, link, target string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		if out, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput(); err != nil {
			t.Fatalf("mklink /J %s %s: %v\n%s", link, target, err, out)
		}
		return
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
}

func mkdirs(t *testing.T, root string, dirs ...string) {
	t.Helper()
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

// ADR-015: a refusal names the fix. In Git Bash an unquoted `parsers\p.py`
// reaches mrw as `parsersp.py`, the shell having taken the backslash as an
// escape. A missing path with no separator that a file under the root spells
// once its separators are removed gets one prose line naming that file; no
// command line is printed, since none runs unchanged in every shell (#591).
func TestAMissingPathAFileSpellsWithoutItsSeparatorsNamesThatFile(t *testing.T) {
	root := realTempDir(t)
	mkdirs(t, root, "parsers", "internal/read", "ab/c", "a/b/c/d")
	writeFile(t, root, "parsers/p.py", "x\n")
	writeFile(t, root, "internal/read/read.go", "package read\n")
	writeFile(t, root, "a/b/c/d/g", "x\n")
	for _, c := range []struct{ spec, want string }{
		{"parsersp.py", "parsers/p.py"},
		{"parsersp.py:1-2", "parsers/p.py"},
		{"internalreadread.go", "internal/read/read.go"},
		{"abcdg", "a/b/c/d/g"},
	} {
		out, problems := run(t, root, Options{Numbers: true}, c.spec)
		if problems != 1 || !strings.Contains(out, "UNREADABLE") {
			t.Fatalf("%s: problems=%d, want UNREADABLE:\n%s", c.spec, problems, out)
		}
		if !strings.Contains(out, "\n    ("+lostSeparator+c.want+"? quote the path, or use /)") {
			t.Errorf("%s: want the hint naming %s:\n%s", c.spec, c.want, out)
		}
		if strings.Contains(out, "mrw read") {
			t.Errorf("%s: the hint prints a command line:\n%s", c.spec, out)
		}
	}
}

// The pair: a missing path that no file spells gets the bare refusal; so does
// one that already holds a separator, an absolute one included (the review of
// this change: load makes it relative before the hint is decided), one a
// DIRECTORY spells, since only a file is a read's answer, one whose file sits
// deeper than eatenSeparatorDepth, and one only a file under .git spells, which
// the walk does not enter either. A link is not followed, to a directory inside
// the root or outside it.
func TestAMissingPathNoFileSpellsGetsNoBackslashHint(t *testing.T) {
	root := realTempDir(t)
	mkdirs(t, root, "ab/c", "parsers", "a/b/c/d/e", ".git")
	writeFile(t, root, "parsers/p.py", "x\n")
	writeFile(t, root, "a/b/c/d/e/f", "x\n")
	writeFile(t, root, ".git/config", "x\n")
	outside := realTempDir(t)
	writeFile(t, outside, "f.txt", "x\n")
	linkDir(t, filepath.Join(root, "out"), outside)
	linkDir(t, filepath.Join(root, "lnk"), filepath.Join(root, "parsers"))
	specs := []string{"nope.py", "abc", "x/parsersp.py", "parsersq.py", "abcdef", ".gitconfig",
		"outf.txt", "lnkp.py", filepath.Join(root, "parsersp.py")}
	for _, spec := range specs {
		assertNoLostSeparatorHint(t, root, spec)
	}
}

// The hint names no file a read of it would refuse: a file inside mrw's own
// state directory (ADR-077) is dropped as a candidate.
func TestTheBackslashHintNamesNoFileAReadWouldRefuse(t *testing.T) {
	root := realTempDir(t)
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "st"))
	mkdirs(t, root, "st/mrw/h")
	writeFile(t, root, "st/mrw/h/root", "x\n")
	assertNoLostSeparatorHint(t, root, "stmrwhroot")
}

// The search lists at most eatenSeparatorLists directories. Every directory
// below holds a, aa and aaa four levels down, and the one answer sits at
// aaa/aaa/aaa/aaa/z, which the prefix-guided search reaches only after more
// than a hundred listings: past the bound it says nothing.
func TestTheBackslashHintStopsAtItsListingBound(t *testing.T) {
	root := realTempDir(t)
	dirs := []string{""}
	for level := 0; level < 4; level++ {
		var next []string
		for _, d := range dirs {
			for _, n := range []string{"a", "aa", "aaa"} {
				next = append(next, filepath.Join(d, n))
			}
		}
		dirs = next
	}
	mkdirs(t, root, dirs...)
	writeFile(t, root, "aaa/aaa/aaa/aaa/z", "x\n")
	assertNoLostSeparatorHint(t, root, "aaaaaaaaaaaaz")
}

// A missing path the hint names is not counted toward the shell-split hint as
// an English word: the two would contradict each other.
func TestAMissingPathTheBackslashHintNamesIsNotAShellSplitWord(t *testing.T) {
	root := realTempDir(t)
	mkdirs(t, root, "parsers")
	writeFile(t, root, "parsers/p", "x\n")
	out, _ := run(t, root, Options{Numbers: true}, "parsersp", "missingword")
	if !strings.Contains(out, lostSeparator+"parsers/p?") {
		t.Fatalf("want the backslash hint for parsersp:\n%s", out)
	}
	if strings.Contains(out, "shell-split regex address") {
		t.Errorf("the shell-split hint contradicts the backslash hint:\n%s", out)
	}
}

func assertNoLostSeparatorHint(t *testing.T, root, spec string) {
	t.Helper()
	out, problems := run(t, root, Options{Numbers: true}, spec)
	if problems != 1 || !strings.Contains(out, "UNREADABLE") {
		t.Fatalf("%s: problems=%d, want UNREADABLE:\n%s", spec, problems, out)
	}
	if strings.Contains(out, "separator may have been lost") {
		t.Errorf("%s: want no backslash hint:\n%s", spec, out)
	}
}
