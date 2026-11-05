package guide

import (
	"regexp"
	"strings"
	"testing"
)

// ADR-185. ADR-184 shortened rule 6's "--stat lists the matching files" to
// "--stat lists files", and in the core-first ablation's runs 3 to 5 an agent
// handed only that core ran a bare `mrw read --stat`, which is refused, in 5 of
// 30 cells, against none of 40 with the old wording or the full guide
// (docs/bench/observability/CORE-RESULTS.md). Rule 6 says which files again,
// rule 1 keeps the sub write, and the core stays under 300 words.
func TestTheCoreSaysStatListsTheMatchingFiles(t *testing.T) {
	core := Core()
	rule := func(n string) string { return regexp.MustCompile(`(?m)^` + n + `\. .*$`).FindString(core) }
	if !strings.Contains(rule("6"), "--stat lists the matching files") {
		t.Errorf("rule 6 does not say --stat lists the matching files:\n%s", rule("6"))
	}
	if !strings.Contains(rule("1"), "mrw write --grep P --sub R") {
		t.Errorf("rule 1 lost the sub write:\n%s", rule("1"))
	}
	if n := len(strings.Fields(core)); n >= 300 {
		t.Errorf("the core is %d words, want under 300", n)
	}
}
