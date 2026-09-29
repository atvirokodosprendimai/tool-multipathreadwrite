package main

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check"
)

// ADR-095 T3. A caller who learns mrw from mrw instructions, AGENTS.md or the
// README learns that a check counts a level as a step does and what is refused
// at the limit, that --no-check writes without a check, what clears the count,
// that a stopped group hears SIGTERM first and what that leaves, and that a
// hanging ast-grep is, on unix, sent SIGTERM at 2 s and killed by 3 s, while on
// Windows, where subproc has no process group to signal, it is killed at 2 s.
// The limit is read from check.MaxStepDepth, so a moved limit turns this red
// instead of leaving the prose behind.
func TestEverySurfaceTeachesTheCheckDepthAndItsLimits(t *testing.T) {
	instructions, code := runIn(t, t.TempDir(), "instructions")
	if code != 0 {
		t.Fatalf("mrw instructions exited %d:\n%s", code, instructions)
	}
	agents, err := os.ReadFile("../../AGENTS.md")
	if err != nil {
		t.Fatal(err)
	}
	readme, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatal(err)
	}
	for name, doc := range map[string]string{
		"mrw instructions": instructions,
		"AGENTS.md":        string(agents),
		"README.md":        string(readme),
	} {
		for _, s := range []string{
			"A check, like a step, runs with MRW_STEP_DEPTH one higher than mrw's own",
			"at depth " + strconv.Itoa(check.MaxStepDepth) + " mrw starts neither",
			"a write whose check is due",
			"a write that starts no check still lands (--no-check writes without it)",
			"mrw check",
			"env -i",
			"sudo",
			"setsid",
			"hears SIGTERM first",
			"killed a second later",
			"can leave its own check running",
			"sent SIGTERM at 2 s",
			"killed by 3 s",
			"on Windows it is killed at 2 s",
		} {
			if !strings.Contains(doc, s) {
				t.Errorf("%s does not teach %q", name, s)
			}
		}
		if strings.Count(doc, "killed at 2 s") != strings.Count(doc, "on Windows it is killed at 2 s") {
			t.Errorf("%s says a hanging ast-grep is killed at 2 s outside its Windows clause", name)
		}
	}
	// ADR-092 T5's guide line: a step's own depth refusal, still taught.
	if s := "--then and --then-sh are refused at depth " + strconv.Itoa(check.MaxStepDepth); !strings.Contains(instructions, s) {
		t.Errorf("mrw instructions does not teach %q", s)
	}
}
