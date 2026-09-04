package main

import (
	"os"
	"strings"
	"testing"
)

// TestEverySubcommandReachesTheAgentFacingGuide derives its universe from the
// CLI's own command table rather than from a list kept beside it.
//
// This is issue #73, and #73 is a RECURRENCE of #51: ADR-007 shipped --grep
// into the README and the contract script but not into AGENTS.md, that was
// found and fixed by hand, and one ADR later ADR-009 shipped `mrw stats` the
// same way. A convention held by intention survived exactly one release, so
// the fix for the second occurrence is a check rather than a third edit — a
// subcommand added tomorrow joins this assertion on the same commit.
//
// AGENTS.md specifically, and not "the docs": README already documents every
// command, so a gate spanning both passes vacuously. The claim being defended
// is that the AGENT-FACING file is complete, because that file is what the
// centralised `mrw` skill mirrors — an agent that loads the skill is taught
// whatever this file omits, and cannot ask for what it cannot see.
//
// Mention is a deliberately weak assertion. Requiring a GOOD explanation is
// not gateable, and a prose-quality gate was measured and rejected in a
// sibling project for that reason. Requiring the backticked invocation to
// appear is crude and cheap, and it would have caught both #51 and #73.
func TestEverySubcommandReachesTheAgentFacingGuide(t *testing.T) {
	b, err := os.ReadFile("../../AGENTS.md")
	if err != nil {
		t.Fatalf("reading the agent-facing guide: %v", err)
	}
	guide := string(b)

	for _, c := range rootCommand().Commands {
		// `help` is the framework's own and is not mrw's surface to teach.
		if c.Name == "help" {
			continue
		}
		// The backticked invocation, not the bare word: "check" and "read"
		// are ordinary English and occur throughout this file in prose, so a
		// bare-word check would pass for a subcommand nobody documented.
		if want := "`mrw " + c.Name; !strings.Contains(guide, want) {
			t.Errorf("AGENTS.md never mentions %q — an agent loading the centralised skill is taught mrw without it (issue #73)", "mrw "+c.Name)
		}
	}
}
