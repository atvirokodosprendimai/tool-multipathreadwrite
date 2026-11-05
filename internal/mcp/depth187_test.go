package mcp

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ADR-187, finding on ADR-095. The CLI's --check is a demand, which makes a
// check due whatever the paths, so the CLI rows in cmd/mrw cannot show that a
// rename's source or destination makes one due. check: true selects ADR-054's
// rule (writer.CheckAuto), which does: at MRW_STEP_DEPTH 8 a write whose check
// is due is refused before anything is written, judged over the plan's own
// paths; a plan with none non-prose, and a write that asks for no check, lands.
func TestAnMCPWriteWhoseCheckIsDueIsRefusedAtTheDepthLimit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the checks are POSIX shell lines")
	}
	t.Setenv("MRW_STEP_DEPTH", "8")
	for _, c := range []struct {
		name string
		plan string
		due  bool
	}{
		{"a .go replace", goEdit113, true},
		{"a .go unlink", "@@ a.go - unlink\n", true},
		{"a .go renamed to .md", "@@ a.go - rename\nb.md\n", true},
		{"a .md renamed to .go", "@@ notes.md - rename\nnotes.go\n", true},
		{"a .md replace", "@@ notes.md 2 replace anchor=\"line two\"\nline 2\n", false},
		{"a .md renamed to .txt", "@@ notes.md - rename\nnotes.txt\n", false},
	} {
		root := checkTree113(t, `{"check":"touch checked"}`)
		res := call(t, root, "mrw_write", map[string]any{"plan": c.plan, "check": true})
		_, statErr := os.Stat(filepath.Join(root, "checked"))
		refused := res["isError"] == true && strings.Contains(firstText(t, res), "MRW_STEP_DEPTH")
		if c.due && statErr == nil {
			t.Errorf("%s: a check ran under a refusal: %v", c.name, res)
		}
		if refused != c.due {
			t.Errorf("%s: refused %v, want %v:\n%s", c.name, refused, c.due, firstText(t, res))
		}
		if c.due {
			if b, _ := os.ReadFile(filepath.Join(root, "a.go")); string(b) != "package a\nfunc A() {}\n" {
				t.Errorf("%s: a.go changed under a refusal: %q", c.name, b)
			}
		}
	}
	root := checkTree113(t, `{"check":"touch checked"}`)
	if res := call(t, root, "mrw_write", map[string]any{"plan": goEdit113}); res["isError"] == true {
		t.Errorf("a write that asks for no check is refused at depth 8: %v", res)
	}
}
