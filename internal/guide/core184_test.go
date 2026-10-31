package guide

import (
	"regexp"
	"slices"
	"strings"
	"testing"
)

// ADR-184. An agent handed only the core is told to make every edit as one plan
// and was not shown what a plan line is: in the core-first ablation the core arm
// fetched the full guide in 8 of its 10 cells to learn it, which cost more than
// carrying the full guide (docs/bench/observability/CORE-RESULTS.md, run 2). Rule
// 4 shows the three headers those jobs needed and where the anchor comes from;
// the full text still names every op and the anchor rule; the core stays under
// 300 words.
func TestTheCoreTeachesTheHunkHeaderAPlanIsMadeOf(t *testing.T) {
	core := Core()
	rule4 := regexp.MustCompile(`(?m)^4\. .*$`).FindString(core)
	// Whole forms, so a mangled op (insert-afterwards) cannot pass as a
	// substring of a valid one (the Codex challenge of ADR-184).
	forms := regexp.MustCompile(`@@ PATH (?:N-M|N) [a-z-]+(?: anchor="TEXT")?`).FindAllString(rule4, -1)
	want := []string{`@@ PATH N-M replace anchor="TEXT"`, `@@ PATH N insert-after`, `@@ PATH N-M delete`}
	if !slices.Equal(forms, want) {
		t.Errorf("rule 4 shows %q, want exactly %q:\n%s", forms, want, rule4)
	}
	if !strings.Contains(rule4, "copied from line N") {
		t.Errorf("rule 4 does not say the anchor is copied from line N:\n%s", rule4)
	}
	for _, want := range []string{"Ops: replace, insert-after, insert-before, delete", "A multi-line replace requires anchor="} {
		if !strings.Contains(CLI(), want) {
			t.Errorf("the full text lost %q", want)
		}
	}
	if n := len(strings.Fields(core)); n >= 300 {
		t.Errorf("the core is %d words, want under 300", n)
	}
}
