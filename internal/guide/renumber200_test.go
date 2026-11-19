package guide

import (
	"strings"
	"testing"
)

// ADR-200 T2. The instructions used to invite the failure the record is about:
// rule 2 and recipe 7 said a file mrw just wrote needs no re-read, and nothing
// said its line numbers had moved. They now say so where they grant the
// licence, and recipe 21 shows the edit that does not depend on a number.
func TestTheInstructionsDoNotInviteAStaleNumber(t *testing.T) {
	core, recipes, cli, skill := Core(), Recipes(), CLI(), Skill("x")

	if !strings.Contains(core, "except in a file mrw just wrote, where a number alone is refused") {
		t.Errorf("rule 2 grants the licence without saying the numbers can move:\n%s", core)
	}
	if n := len(strings.Fields(core)); n >= 300 {
		t.Errorf("the core is %d words, want under 300", n)
	}

	var seven, twentyOne, command string
	lines := strings.Split(recipes, "\n")
	for i, l := range lines {
		switch {
		case strings.HasPrefix(l, "7. "):
			seven = l
		case strings.HasPrefix(l, "21. "):
			twentyOne = l
			if i+1 < len(lines) {
				command = lines[i+1]
			}
		}
	}
	if strings.Contains(seven, "needs no re-read") || !strings.Contains(seven, "until a read has shown a line a number alone is refused") {
		t.Errorf("recipe 7 still says a written file needs no re-read, or does not point at recipe 21:\n%s", seven)
	}
	if !strings.Contains(seven, "keeps a whole-file licence, unless every hunk on it was a search_replace block (the write then adds the lines it wrote to the licence it had)") {
		t.Errorf("recipe 7 does not say what licence a written file keeps:\n%s", seven)
	}
	if !strings.Contains(twentyOne, "a file you just wrote") || !strings.Contains(twentyOne, "a number alone is refused, unless a read has shown that line since the write, so name the line by a pattern") || !strings.Contains(twentyOne, "or read the line first") || strings.Contains(twentyOne, "only after") {
		t.Errorf("there is no recipe 21 for a second edit to a file just written:\n%s", twentyOne)
	}
	// The command names its line by content twice: a pattern address, and an
	// anchor. A number in it would be the thing the recipe warns about.
	if !strings.HasPrefix(command, "  ") || !strings.Contains(command, "@@ f.go /") || !strings.Contains(command, `anchor="`) || !strings.Contains(command, "mrw write -") {
		t.Errorf("recipe 21's command does not address by pattern with an anchor:\n%q", command)
	}
	for name, text := range map[string]string{"CLI": cli, "Skill": skill} {
		if !strings.Contains(text, twentyOne) || !strings.Contains(text, command) {
			t.Errorf("%s does not carry recipe 21", name)
		}
	}
}
