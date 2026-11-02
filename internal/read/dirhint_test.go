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
