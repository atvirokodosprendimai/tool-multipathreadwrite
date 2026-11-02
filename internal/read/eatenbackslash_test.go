package read

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ADR-015: a refusal names the fix. In Git Bash an unquoted `parsers\p.py`
// reaches mrw as `parsersp.py`, the shell having taken the backslash as an
// escape. A missing path with no separator that a file under the root spells
// once its separators are removed gets one prose line naming that file; no
// command line is printed, since none runs unchanged in every shell (#591).
func TestAMissingPathAFileSpellsWithoutItsSeparatorsNamesThatFile(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{"parsers", "internal/read", "ab/c", "a/b/c/d"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, root, "parsers/p.py", "x\n")
	writeFile(t, root, "internal/read/read.go", "package read\n")
	writeFile(t, root, "a/b/c/d/g", "x\n")
	const eaten = "a shell may have eaten a backslash: did you mean "
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
		if !strings.Contains(out, "\n    ("+eaten+c.want+"? quote the path, or use /)") {
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
// DIRECTORY spells, since only a file is a read's answer, and one whose file sits
// deeper than eatenSeparatorDepth. A link is not followed.
func TestAMissingPathNoFileSpellsGetsNoBackslashHint(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{"ab/c", "parsers", "a/b/c/d/e"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, root, "parsers/p.py", "x\n")
	writeFile(t, root, "a/b/c/d/e/f", "x\n")
	specs := []string{"nope.py", "abc", "x/parsersp.py", "parsersq.py", "abcdef", filepath.Join(root, "parsersp.py")}
	if runtime.GOOS != "windows" {
		outside := t.TempDir()
		writeFile(t, outside, "f.txt", "x\n")
		if err := os.Symlink(outside, filepath.Join(root, "lnk")); err != nil {
			t.Fatal(err)
		}
		specs = append(specs, "lnkf.txt")
	}
	for _, spec := range specs {
		out, problems := run(t, root, Options{Numbers: true}, spec)
		if problems != 1 || !strings.Contains(out, "UNREADABLE") {
			t.Fatalf("%s: problems=%d, want UNREADABLE:\n%s", spec, problems, out)
		}
		if strings.Contains(out, "eaten a backslash") {
			t.Errorf("%s: want no backslash hint:\n%s", spec, out)
		}
	}
}
