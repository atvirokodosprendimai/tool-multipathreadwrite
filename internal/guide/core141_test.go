package guide

import (
	"regexp"
	"strings"
	"testing"
)

// ADR-141. The short form an agent is handed: eight numbered rules in under 300
// words, the last line naming the whole contract, and every rule something the
// full contract also teaches, so the full text cannot lose a rule the core
// keeps.
func TestTheCoreIsEightShortRulesTheFullContractStillCarries(t *testing.T) {
	core := Core()
	if n := len(strings.Fields(core)); n > 300 {
		t.Errorf("the core is %d words, want at most 300", n)
	}
	rules := regexp.MustCompile(`(?m)^[1-8]\. `).FindAllString(core, -1)
	if len(rules) != 8 {
		t.Errorf("the core has %d numbered rules, want 8:\n%s", len(rules), core)
	}
	if !strings.Contains(core, "mrw instructions prints the whole contract") {
		t.Errorf("the core does not name the full form:\n%s", core)
	}
	full := CLI()
	for rule, marker := range map[int]string{
		1: "one read of every site",
		2: "read on past the range",
		3: "nothing is written",
		4: "body= is a line count",
		5: "Never read an exit code through a pipe",
		6: "(?i)",
		7: "mrw write --create PATH",
		8: "before the subcommand",
	} {
		if !strings.Contains(full, marker) {
			t.Errorf("rule %d: the full contract no longer carries %q, which the core teaches", rule, marker)
		}
	}
	for _, marker := range []string{"--max-cols", "--stat", "--no-numbers", "(?i)", "anchor=", "--create"} {
		if !strings.Contains(core, marker) {
			t.Errorf("the core does not teach %q", marker)
		}
	}
}
