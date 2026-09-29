package main

import (
	"os"
	"strings"
	"testing"
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
			"A step runs as written: a step command holding {files} or {packages} is refused, since mrw expands them only in scoped_check.",
			"A passing step prints the last line of its output under its verdict.",
		} {
			if !strings.Contains(doc, s) {
				t.Errorf("%s does not teach %q", name, s)
			}
		}
	}
}
