package guide

import (
	"strings"
	"testing"
)

// ADR-169 Decision 6: the re-read guard is taught where callers look — the
// full instructions name the flag, the recipes show it, so the skill carries
// it — and the core takes it only within ADR-141's bound.
func TestTheInstructionsTeachTheReReadGuard(t *testing.T) {
	const flag = "--if-changed PATH=SHA"
	const recipe = "mrw read --if-changed f.go=8dc37337 f.go:40-60"
	if cli := CLI(); !strings.Contains(cli, flag) || !strings.Contains(cli, recipe) {
		t.Errorf("mrw instructions does not teach the guard: want %q and the recipe %q", flag, recipe)
	}
	if s := Skill("v9.9.9"); !strings.Contains(s, recipe) {
		t.Errorf("the skill does not carry the guard's recipe %q", recipe)
	}
	if n := len(strings.Fields(Core())); n > 300 {
		t.Errorf("the core is %d words, over ADR-141's 300", n)
	}
}
