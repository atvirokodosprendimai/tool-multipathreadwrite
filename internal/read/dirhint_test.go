package read

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-015: a refusal names the fix. A directory named to read is UNREADABLE, as
// before, and the line after it says how to read one: walk it with --grep. In
// the benchmark's run 8 an agent's `mrw read .` cost a turn on "is a directory"
// alone. The pair: a missing file gets no such line.
func TestADirectoryNamedToReadSaysHowToWalkIt(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "sub/a.txt", "a\n")
	for _, c := range []struct {
		spec string
		hint bool
	}{{"sub", true}, {"sub:1-2", true}, {"nope.txt", false}} {
		out, problems := run(t, root, Options{Numbers: true}, c.spec)
		if problems != 1 || !strings.Contains(out, "UNREADABLE") {
			t.Fatalf("%s: problems=%d, want the UNREADABLE refusal:\n%s", c.spec, problems, out)
		}
		got := strings.Contains(out, "a directory you name is walked: mrw read --grep PATTERN sub")
		if got != c.hint {
			t.Errorf("%s: hint present %v, want %v:\n%s", c.spec, got, c.hint, out)
		}
	}
}
