package guide

import (
	"strings"
	"testing"
)

// ADR-178 T2. The recipes, which `mrw instructions` and `mrw skill` both print,
// teach `--json-path` on standard input, say that a projection licenses
// nothing, and name the path address to read before editing the value.
func TestTheGuideTeachesJSONPathProjection(t *testing.T) {
	for name, text := range map[string]string{"Recipes": Recipes(), "Skill": Skill("x")} {
		// The words are judged inside the recipe itself: recipe 7 already says
		// "licenses nothing" of a read into /dev/null.
		i := strings.Index(text, "\n13. ")
		if i < 0 {
			t.Errorf("%s has no recipe 13", name)
			continue
		}
		recipe := text[i:]
		for _, want := range []string{"| mrw read --json-path '$.a,$.b' -", "licenses nothing", "f.json:$.a.b"} {
			if !strings.Contains(recipe, want) {
				t.Errorf("%s: recipe 13 does not hold %q", name, want)
			}
		}
	}
}
