package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/guide"
)

// ADR-200 T2. Recipe 21 is the edit to a file mrw just wrote that needs no
// line number. The command the instructions print is run as printed, by a
// shell, against the built binary: after a write that moved func B down a
// line, with func B never read, it changes func B and nothing else. A recipe
// whose command does nothing, or whose pattern, anchor or body is wrong,
// fails here, where a check of its wording cannot.
func TestRecipeTwentyOneAppliesToAFileJustWritten(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the recipe is run by a POSIX shell")
	}
	var command string
	lines := strings.Split(guide.Recipes(), "\n")
	for i, l := range lines {
		if strings.HasPrefix(l, "21. ") && i+1 < len(lines) {
			command = strings.TrimSpace(lines[i+1])
		}
	}
	if command == "" {
		t.Fatal("there is no recipe 21")
	}

	bindir := t.TempDir()
	if out, err := exec.Command("go", "build", "-o", filepath.Join(bindir, "mrw"), ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	root, state := t.TempDir(), t.TempDir()
	sh := func(line string) (string, int) {
		t.Helper()
		c := exec.Command("sh", "-c", line)
		c.Dir = root
		c.Env = append(os.Environ(), "PATH="+bindir+string(os.PathListSeparator)+os.Getenv("PATH"), "XDG_STATE_HOME="+state)
		out, err := c.CombinedOutput()
		code := 0
		if err != nil {
			code = c.ProcessState.ExitCode()
		}
		return string(out), code
	}
	writeFiles(t, root, map[string]string{"f.go": "package a\n\nfunc A() {\n}\n\nfunc B() {\n}\n"})
	// Only the lines the first write needs are read: func B is licensed by
	// that write alone, which is the licence the recipe is about.
	if out, code := sh("mrw read f.go:1-3"); code != 0 {
		t.Fatalf("read: exit %d:\n%s", code, out)
	}
	if out, code := sh(`printf '%s\n' '@@ f.go 2 insert-after' '// added' | mrw write -`); code != 0 {
		t.Fatalf("the write that moves func B down: exit %d:\n%s", code, out)
	}
	if out, code := sh(command); code != 0 {
		t.Fatalf("recipe 21 as printed: exit %d:\n%s", code, out)
	}
	got, err := os.ReadFile(filepath.Join(root, "f.go"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "package a\n\n// added\nfunc A() {\n}\n\nfunc B(x int) {\n}\n"; string(got) != want {
		t.Errorf("after recipe 21:\n%s\nwant:\n%s", got, want)
	}
}
