package guide

import (
	"strings"
	"testing"
)

// ADR-179 T9. The recipes — printed by `mrw instructions` and by `mrw skill` —
// teach each shorter read and the terse receipt, and never teach --if-served
// without the condition that makes it safe.
func TestTheGuideTeachesTheShorterReads(t *testing.T) {
	recipes, skill := Recipes(), Skill("x")
	for _, want := range []string{"--outline", "--max-matches", "--fold", "--no-comments", "--if-served", "--terse", "MRW_MAX_COLS"} {
		if !strings.Contains(recipes, want) {
			t.Errorf("Recipes() never names %s", want)
		}
		if !strings.Contains(skill, want) {
			t.Errorf("the skill never names %s", want)
		}
	}
	var served string
	for _, r := range strings.Split(recipes, "\n") {
		if strings.Contains(r, "--if-served") && strings.Contains(r, "seen --forget") {
			served = r
		}
	}
	if served == "" {
		t.Fatalf("no recipe line names --if-served with seen --forget:\n%s", recipes)
	}
	for _, cond := range []string{"alone in its checkout", "start, resume, clear and compaction", "--max-chars"} {
		if !strings.Contains(served, cond) {
			t.Errorf("the --if-served recipe lacks its condition %q:\n%s", cond, served)
		}
	}
}
