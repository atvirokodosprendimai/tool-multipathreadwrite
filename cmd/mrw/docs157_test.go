package main

import (
	"os"
	"strings"
	"testing"
)

// ADR-157 Decision 8, from the review of #412: each doc names every shell's
// form for a JSON path beside that shell, so dropping one recipe, or moving it
// to another shell, fails here. Line breaks are folded first.
func TestTheDocsQuoteAJSONPathPerShell(t *testing.T) {
	pairs := []string{
		"pwsh 7 takes `'t.json:$[\"k\"]'`",
		"Windows PowerShell 5.1 needs `'t.json:$[\\\"k\\\"]'`",
		"cmd needs `\"t.json:$[\\\"k\\\"]\"`",
		"Git Bash",
	}
	for _, doc := range []string{"../../AGENTS.md", "../../README.md"} {
		b, err := os.ReadFile(doc)
		if err != nil {
			t.Fatal(err)
		}
		text := strings.Join(strings.Fields(string(b)), " ")
		for _, p := range pairs {
			if !strings.Contains(text, p) {
				t.Errorf("%s does not say %q", doc, p)
			}
		}
	}
}
