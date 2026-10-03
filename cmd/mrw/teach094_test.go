package main

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"
)

// ADR-094 T3. A caller who learns mrw from mrw instructions, AGENTS.md or the
// README learns that a step runs as written, so a step command holding
// scoped_check's tokens is refused, and that a passing step prints its last
// line. The instructions are read from the command's own output.
func TestEverySurfaceTeachesWhatAStepChecked(t *testing.T) {
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
			"A step runs as written: a step command holding {files}, {dirs} or {packages} is refused, since mrw expands them only in scoped_check.",
			"A passing step prints the last line of its output under its verdict.",
		} {
			if !strings.Contains(doc, s) {
				t.Errorf("%s does not teach %q", name, s)
			}
		}
	}
}

// ADR-094 T3. The instructions name no flag the CLI lacks: every --name in the
// output of mrw instructions is a flag some mrw command declares, so a sentence
// added to teach a step cannot teach a flag that does not exist. Contract §115
// holds the stricter rule on the built binary; this binds the claim in T3's
// fence.
func TestTheInstructionsNameNoFlagTheCLILacks(t *testing.T) {
	instructions, code := runIn(t, t.TempDir(), "instructions")
	if code != 0 {
		t.Fatalf("mrw instructions exited %d:\n%s", code, instructions)
	}
	known := map[string]bool{"help": true}
	var walk func(c *cli.Command)
	walk = func(c *cli.Command) {
		for _, f := range c.Flags {
			for _, n := range f.Names() {
				known[n] = true
			}
		}
		for _, sub := range c.Commands {
			walk(sub)
		}
	}
	walk(rootCommand())
	for _, m := range regexp.MustCompile(`--([a-z][a-z-]*)`).FindAllStringSubmatch(instructions, -1) {
		if !known[m[1]] {
			t.Errorf("mrw instructions names --%s, which no mrw command declares", m[1])
		}
	}
}
