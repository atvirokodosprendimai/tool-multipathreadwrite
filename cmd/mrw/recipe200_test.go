package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/guide"
)

// ADR-200 T2. Recipe 21 is the edit to a file mrw just wrote that needs no
// line number. The command printed in the instructions is taken apart and its
// plan applied for real: after a write that moved func B down a line, the
// recipe's hunk changes func B and nothing else. A recipe whose pattern or
// anchor is broken fails here, where a check of its wording cannot.
func TestRecipeTwentyOneAppliesToAFileJustWritten(t *testing.T) {
	var command string
	lines := strings.Split(guide.Recipes(), "\n")
	for i, l := range lines {
		if strings.HasPrefix(l, "21. ") && i+1 < len(lines) {
			command = lines[i+1]
		}
	}
	// printf '%s\n' 'HEADER' 'BODY' | mrw write -
	quoted := regexp.MustCompile(`'([^']*)'`).FindAllStringSubmatch(command, -1)
	if len(quoted) != 3 || quoted[0][1] != `%s\n` || !strings.HasSuffix(command, "| mrw write -") {
		t.Fatalf("recipe 21 is not printf of a header and one body line piped to mrw write -:\n%q", command)
	}
	header, body := quoted[1][1], quoted[2][1]

	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	writeFiles(t, root, map[string]string{"f.go": "package a\n\nfunc A() {\n}\n\nfunc B() {\n}\n"})
	if out, code := runIn(t, root, "read", "f.go"); code != 0 {
		t.Fatalf("read: exit %d:\n%s", code, out)
	}
	if out, code := writeIn(t, root, "--no-check", planFile(t, "@@ f.go 2 insert-after\n// added\n")); code != 0 {
		t.Fatalf("the write that moves func B down: exit %d:\n%s", code, out)
	}
	// No read between: the file is mrw's own write, and func B is now line 7.
	if out, code := writeIn(t, root, "--no-check", planFile(t, header+"\n"+body+"\n")); code != 0 {
		t.Fatalf("recipe 21's plan: exit %d:\n%s", code, out)
	}
	got, err := os.ReadFile(filepath.Join(root, "f.go"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "package a\n\n// added\nfunc A() {\n}\n\n" + body + "\n}\n"; string(got) != want {
		t.Errorf("after recipe 21:\n%s\nwant:\n%s", got, want)
	}
}
