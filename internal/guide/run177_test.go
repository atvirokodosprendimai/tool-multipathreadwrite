package guide

import (
	"strings"
	"testing"
)

// ADR-177 T2. mrw run is taught where an agent looks first: the recipes, which
// mrw instructions and mrw skill both print. The recipe carries the form that
// reads the log by pattern from standard input, the exit-code sentence, and
// the one trust caveat; and it pipes nothing.
func TestTheGuideTeachesMrwRun(t *testing.T) {
	for name, text := range map[string]string{"Recipes": Recipes(), "CLI": CLI(), "Skill": Skill("x")} {
		for _, want := range []string{"mrw run '", "- < '", ThenShCaveat()} {
			if !strings.Contains(text, want) {
				t.Errorf("%s lacks %q", name, want)
			}
		}
	}
	if !strings.Contains(CLI(), "128+N") || !strings.Contains(CLI(), "124") || !strings.Contains(CLI(), "130") {
		t.Errorf("mrw instructions does not give mrw run's exit codes")
	}
	// No recipe line that runs mrw run sends its answer through a pipe, which
	// would read the pipe's status for the command's.
	for _, l := range strings.Split(Recipes(), "\n") {
		if strings.Contains(l, "mrw run '") && strings.Contains(l, "|") {
			t.Errorf("a recipe pipes mrw run: %q", l)
		}
	}
}
