package mcp

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ADR-187, finding on ADR-095, ADR-205. check: true makes a check due after any
// write that lands, as the CLI's --check does, so at MRW_STEP_DEPTH 8 every
// write that asks for it is refused before anything is written, a prose-only
// one and a rename between prose names included; a write that asks for no
// check lands.
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
		{"a .md replace", "@@ notes.md 2 replace anchor=\"line two\"\nline 2\n", true},
		{"a .md renamed to .txt", "@@ notes.md - rename\nnotes.txt\n", true},
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
			// Every file the plan names is as the fixture made it: a.go and
			// notes.md still in place and unedited, and no renamed copy.
			for name, want := range map[string]string{"a.go": "package a\nfunc A() {}\n", "notes.md": "# notes\nline two\n"} {
				if b, err := os.ReadFile(filepath.Join(root, name)); err != nil || string(b) != want {
					t.Errorf("%s: %s changed or moved under a refusal: %q, %v", c.name, name, b, err)
				}
			}
			for _, moved := range []string{"b.md", "notes.go", "notes.txt"} {
				if _, err := os.Stat(filepath.Join(root, moved)); err == nil {
					t.Errorf("%s: %s exists after a refusal", c.name, moved)
				}
			}
		}
	}
	root := checkTree113(t, `{"check":"touch checked"}`)
	if res := call(t, root, "mrw_write", map[string]any{"plan": goEdit113}); res["isError"] == true {
		t.Errorf("a write that asks for no check is refused at depth 8: %v", res)
	}
}
