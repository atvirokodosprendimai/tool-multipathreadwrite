package read

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ADR-015: a refusal names the fix. A directory named to read is UNREADABLE, as
// before, and the line after it says how to read one: walk it with --grep, in a
// command that runs as printed. In the benchmark's run 8 an agent's `mrw read .`
// cost a turn on "is a directory" alone. A missing file, or a file that exists
// and cannot be read, gets no such line (the go-reviewer pass found the second
// case untested).
func TestADirectoryNamedToReadSaysHowToWalkIt(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{"sub", "my dir"} {
		if err := os.Mkdir(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, root, d+"/a.txt", "a\n")
	}
	cases := []struct{ spec, hint string }{
		{"sub", "mrw read --grep PATTERN -- sub serves"},
		{"sub:1-2", "mrw read --grep PATTERN -- sub serves"},
		{"my dir", "mrw read --grep PATTERN -- 'my dir' serves"},
		{"nope.txt", ""},
	}
	if runtime.GOOS != "windows" {
		if err := os.Symlink("sub", filepath.Join(root, "link")); err != nil {
			t.Fatal(err)
		}
		cases = append(cases, struct{ spec, hint string }{"link", "mrw read --grep PATTERN -- sub serves"})
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
		walk := strings.Contains(out, "a directory you name is walked")
		if c.hint == "" && walk || c.hint != "" && !strings.Contains(out, c.hint) {
			t.Errorf("%s: want hint %q:\n%s", c.spec, c.hint, out)
		}
		if c.hint != "" && !strings.Contains(out, "--grep $ -- ") {
			t.Errorf("%s: the every-line form is not --grep $:\n%s", c.spec, out)
		}
	}
}

// hintWord quotes for the caller's shell: on Windows a path that needs quoting is
// double-quoted, which cmd.exe groups and single quotes are not (measured on
// Windows 11 by a peer session, 2026-11-02); one holding a character cmd or pwsh
// reads inside double quotes keeps quoteWord's form; elsewhere it is quoteWord's.
func TestAHintQuotesAPathForEveryShell(t *testing.T) {
	for _, c := range []struct {
		name    string
		windows bool
		want    string
	}{
		{"sub", true, "sub"},
		{"my dir", true, "\"my dir\""},
		{"a$b c", true, quoteWord("a$b c", true)},
		{"my dir", false, quoteWord("my dir", false)},
	} {
		if got := hintWord(c.name, c.windows); got != c.want {
			t.Errorf("hintWord(%q, %v) = %q, want %q", c.name, c.windows, got, c.want)
		}
	}
}
