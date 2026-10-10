package guide

import (
	"regexp"
	"strings"
	"testing"
)

// coreRule is one of the eight rules: the clauses its line in Core() must carry,
// and the clauses the full contract must carry so that it cannot lose what the
// core teaches (ADR-141). Both sides are asserted per rule, not as one loose
// marker, and a clause is the sentence's essential part.
var coreRules = []struct {
	n    int
	core []string
	full []string
}{
	{1, []string{"every file read, edit and create", "one read of every site", "mrw write -"},
		[]string{"one read of every site", "mrw write - reads it from standard input"}},
	{2, []string{"per line", "except in a file mrw just wrote", "read on past your range"},
		[]string{"Read before you write, per line", "A file mrw just wrote is wholly known", "read on past the range"}},
	{3, []string{"fails validation writes nothing", "--force", "reports what reached disk"},
		[]string{"nothing is written", "reports what reached disk", "PARTIALLY APPLIED"}},
	{4, []string{"anchor=", "body= is a line count"},
		[]string{"A multi-line replace requires anchor=", "body= is a line count"}},
	{5, []string{"For a write: exit 0", "1 a hunk failed and nothing was written", "3 applied but the check did not pass", "A read exits 1", "through a pipe"},
		[]string{"A write exits 1 when a hunk fails validation", "Exit 3 means the write applied and the check did not pass", "Never read an exit code through a pipe"}},
	{6, []string{"(?i)", "--stat lists", "--max-cols", "unread", "--no-numbers"},
		[]string{"start it with (?i)", "--stat prints only length, size and sha", "--max-cols N (CLI) cuts a line to a window round the match and does not record it as read", "--no-numbers drops the numbers"}},
	{7, []string{"mrw write --create PATH < content", "@@ path 0 create"},
		[]string{"mrw write --create PATH < content", "@@ path 0 create makes a new file"}},
	{8, []string{"BEFORE the subcommand", "-C N is context lines", "after --"},
		[]string{"before the subcommand", "goes after --"}},
}

// ADR-141. The short form an agent is handed: exactly eight numbered rules in
// under 300 words, the first line naming the whole contract, every rule's
// essential clauses present in the core AND in the full contract.
func TestTheCoreIsEightShortRulesTheFullContractStillCarries(t *testing.T) {
	core, full := Core(), CLI()
	if n := len(strings.Fields(core)); n > 300 {
		t.Errorf("the core is %d words, want at most 300", n)
	}
	if !strings.HasPrefix(core, "The eight rules that matter most (mrw instructions prints the whole contract)") {
		t.Errorf("the core's first line does not name the full form:\n%s", core)
	}
	num := regexp.MustCompile(`(?m)^([0-9]+)\. `).FindAllStringSubmatch(core, -1)
	if len(num) != 8 {
		t.Fatalf("the core has %d numbered rules, want exactly 8:\n%s", len(num), core)
	}
	for i, m := range num {
		if m[1] != string(rune('1'+i)) {
			t.Errorf("rule %d is numbered %s, want %d", i+1, m[1], i+1)
		}
	}
	for _, r := range coreRules {
		line := regexp.MustCompile(`(?m)^` + string(rune('0'+r.n)) + `\. .*$`).FindString(core)
		for _, c := range r.core {
			if !strings.Contains(line, c) {
				t.Errorf("rule %d of the core lost the clause %q:\n%s", r.n, c, line)
			}
		}
		for _, c := range r.full {
			if !strings.Contains(full, c) {
				t.Errorf("rule %d: the full contract lost %q, which the core teaches", r.n, c)
			}
		}
	}
}
