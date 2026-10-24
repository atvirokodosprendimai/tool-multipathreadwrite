package guide

import (
	"strings"
	"testing"
)

// ADR-171 Decision 1. The core stops teaching a count for every body: rule 4
// says to leave body= out, and the clause agents read as an instruction is gone.
func TestTheCoreTeachesTheBodyCountAsOptional(t *testing.T) {
	core := Core()
	if !strings.Contains(core, "omit body= unless mrw instructions names your case") {
		t.Errorf("the core does not say to omit body=:\n%s", core)
	}
	if strings.Contains(core, "body= is a line count and goes on the header") {
		t.Errorf("the core still teaches a count for every body:\n%s", core)
	}
}

// ADR-171 Decision 1. "Unless mrw instructions names your case" is only true if
// the full text names every case the parser needs a count for, and what a count
// still buys.
func TestTheFullTextNamesEveryCaseThatNeedsACount(t *testing.T) {
	full := CLI()
	for _, want := range []string{
		"body= is optional for an ordinary inline body",
		"an empty create is body=0",
		"a body holding a line that begins @@ needs body=N",
		"raw=true too when that line is a valid hunk header",
		"a body whose first line begins body= needs its count",
		"from= and body=@path take no inline count",
		"it cannot catch a hunk left out whole",
	} {
		if !strings.Contains(full, want) {
			t.Errorf("the full text lost %q", want)
		}
	}
}

// ADR-171 Decision 2. The header says when to read the rules, not that the
// whole contract follows them.
func TestTheCoreSaysWhenToReRead(t *testing.T) {
	core := Core()
	if !strings.HasPrefix(core, "The eight rules that matter most; read them once a session, and after a compaction only if lost:") {
		t.Errorf("the core's header does not say when to re-read:\n%s", core)
	}
	if strings.Contains(core, "prints the whole contract") {
		t.Errorf("the core's header still sends the reader to the full text:\n%s", core)
	}
	if !strings.Contains(CLI(), "After a compaction, re-read the lines you will edit") {
		t.Errorf("the full text does not say to re-read the lines to be edited after a compaction")
	}
}

// ADR-171 Decision 3. Recipe 5 sends the plan in a file, so no shell parser
// sees its text, and no recipe pipes a plan variable.
func TestRecipeFiveSendsThePlanInAFile(t *testing.T) {
	for name, text := range map[string]string{"Recipes": Recipes(), "CLI": CLI()} {
		if !strings.Contains(text, `mrw write "$TMPDIR/plan.txt"`) {
			t.Errorf("%s: recipe 5 does not write a plan file", name)
		}
		if strings.Contains(text, `"$plan" | mrw write -`) {
			t.Errorf("%s: a recipe still pipes a plan variable through the shell", name)
		}
	}
	if !strings.Contains(Core(), "mrw write PLANFILE") {
		t.Errorf("rule 1 does not name mrw write PLANFILE:\n%s", Core())
	}
}
