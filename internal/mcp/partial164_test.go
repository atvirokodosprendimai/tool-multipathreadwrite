package mcp

import (
	"runtime"
	"strings"
	"testing"
)

// ADR-164 Decision 4 over MCP. A write that places some files runs those, and
// the text names the rest; under a ceiling too small for the receipt the list
// goes with the structured value but the terminal phrase keeps the count.
func TestTheMCPTextKeepsTheCountOutsideTheScopedCheck(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the checks are POSIX shell lines")
	}
	const harness = `{"check":"echo WHOLE","scoped_check":"echo {packages}"}`
	const plan = goEdit113 + "@@ run.sh 1 replace\nfalse\n"
	tree := func(t *testing.T) string {
		t.Helper()
		files := map[string]string{"a.go": "package a\nfunc A() {}\n", "run.sh": "true\n", ".quality-harness.json": harness}
		return licensed(t, files)
	}

	t.Run("the text and the receipt name the file", func(t *testing.T) {
		root := tree(t)
		res := call(t, root, "mrw_write", map[string]any{"plan": plan, "check": true})
		chk, _ := structured(t, res)["check"].(map[string]any)
		if chk == nil || chk["command"] != "echo ." {
			t.Fatalf("check %v, want the scoped command over the placed package", chk)
		}
		if out, _ := chk["outside_scope"].([]any); len(out) != 1 || out[0] != "run.sh" {
			t.Errorf("check.outside_scope = %v, want [run.sh]", chk["outside_scope"])
		}
		want := `outside the scoped check: run.sh — scoped_check "echo {packages}" runs {packages}, which names Go packages only`
		if text := firstText(t, res); !strings.Contains(text, want+"\n") {
			t.Errorf("the text lacks %q:\n%s", want, text)
		}
	})

	t.Run("a write that places everything says nothing", func(t *testing.T) {
		root := tree(t)
		res := call(t, root, "mrw_write", map[string]any{"plan": goEdit113, "check": true})
		if text := firstText(t, res); strings.Contains(text, "outside the scoped check") {
			t.Errorf("nothing was outside, and the text says so:\n%s", text)
		}
	})

	t.Run("the terminal phrase keeps the count", func(t *testing.T) {
		restore := MaxResultChars
		MaxResultChars = minWriteCeiling()
		t.Cleanup(func() { MaxResultChars = restore })
		root := tree(t)
		res := call(t, root, "mrw_write", map[string]any{"plan": plan, "check": true})
		text := firstText(t, res)
		if !strings.Contains(text, "Its check passed for the packages it could place; 1 file(s) were outside it") {
			t.Fatalf("at ceiling %d the answer lost the count:\n%s", MaxResultChars, text)
		}
		if strings.Contains(text, "run.sh") && strings.Contains(text, "outside the scoped check:") {
			t.Errorf("the list survived a ceiling that dropped the receipt:\n%s", text)
		}
	})

	t.Run("a failed check keeps the count too", func(t *testing.T) {
		restore := MaxResultChars
		MaxResultChars = minWriteCeiling()
		t.Cleanup(func() { MaxResultChars = restore })
		files := map[string]string{"a.go": "package a\nfunc A() {}\n", "run.sh": "true\n",
			".quality-harness.json": `{"check":"echo WHOLE","scoped_check":"false {packages}"}`}
		res := call(t, licensed(t, files), "mrw_write", map[string]any{"plan": plan, "check": true})
		text := firstText(t, res)
		if !strings.Contains(text, "Its check did not pass") || !strings.Contains(text, "1 file(s) were outside the scoped check") {
			t.Fatalf("at ceiling %d a failed check lost the count:\n%s", MaxResultChars, text)
		}
	})
}
