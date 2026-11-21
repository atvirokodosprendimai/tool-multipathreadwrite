package mcp

import (
	"strings"
	"testing"
)

// #649, the MCP twin of the CLI test: the text rows of a failed and a skipped path op
// name the address the plan wrote, `-`, not the unresolved bound.
func TestAPathOpRowOfTheMCPAnswerPrintsTheAddressThePlanWrote(t *testing.T) {
	root := licensed(t, map[string]string{"a.txt": "a\n"})
	res := call(t, root, "mrw_write", map[string]any{"plan": "@@ a.txt - unlink\n@@ nope.txt - unlink\n"})
	text := resultText(res)
	for _, want := range []string{"failed nope.txt - ", "skipped a.txt - "} {
		if !strings.Contains(text, want) {
			t.Errorf("want a row starting %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, " 0 ") && (strings.Contains(text, "nope.txt 0") || strings.Contains(text, "a.txt 0")) {
		t.Errorf("a row names address 0:\n%s", text)
	}
}
