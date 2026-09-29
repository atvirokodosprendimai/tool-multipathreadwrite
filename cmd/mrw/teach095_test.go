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
// at the limit, what clears the count, that a stopped group hears SIGTERM
// first and what that leaves, and that a hanging ast-grep is sent SIGTERM at
// 2 s and killed by 3 s. The limit is read from check.MaxStepDepth, so a moved
// limit turns this red instead of leaving the prose behind.
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
			"mrw check",
			"env -i",
			"sudo",
			"setsid",
			"hears SIGTERM first",
			"killed a second later",
			"can leave its own check running",
			"sent SIGTERM at 2 s",
			"killed by 3 s",
		} {
			if !strings.Contains(doc, s) {
				t.Errorf("%s does not teach %q", name, s)
			}
		}
		if strings.Contains(doc, "killed at 2 s") {
			t.Errorf("%s still says a hanging ast-grep is killed at 2 s", name)
		}
	}
}
