package guide

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestEverySurfaceContainsTheSharedSentences(t *testing.T) {
	got := Shared()
	for _, must := range []string{
		"Use mrw always: plan the activity as one read of every site, then one plan, then one write.",
		"nothing is written",
		"per line, not per file",
		"models no target syntax",
		"names the file, the plan line, and the reason",
	} {
		if !strings.Contains(got, must) {
			t.Errorf("Shared() does not teach %q:\n%s", must, got)
		}
	}
	if strings.Contains(got, "@@") {
		t.Error("Shared() embeds a plan example; examples stay in the MCP file that executes them")
	}
	if strings.Contains(got, WhyAllOrNothing()) {
		t.Error("Shared() absorbed the why; it stays extra on CLI() and the handshake")
	}
}

func TestCLIContainsSharedAndTheOperatorTraps(t *testing.T) {
	got := CLI()
	if !strings.Contains(got, Shared()) {
		t.Error("CLI() does not contain Shared() verbatim")
	}
	for _, must := range []string{
		WhyAllOrNothing(),
		"through a pipe",
		"Exit 3",
		"MSYS",
		"glob",
		`anchor="`,
		"single-quot",
		"body=",
		"line count",
		"lines=",
		"-C",
		"--root",
		"Python",
	} {
		if !strings.Contains(got, must) {
			t.Errorf("CLI() does not teach %q:\n%s", must, got)
		}
	}
}

func TestCLITeachesAlwaysAndAPlan(t *testing.T) {
	got := CLI()
	for _, must := range []string{
		"Use mrw always",
		"one read of every site, then one plan, then one write",
		"@@ path 0 create",
	} {
		if !strings.Contains(got, must) {
			t.Errorf("CLI() does not teach %q:\n%s", must, got)
		}
	}
	if strings.Contains(got, "3 or more edits") {
		t.Error("CLI still teaches the threshold that trains agents never to use mrw")
	}
	if strings.Contains(Shared(), "@@ path 0 create") {
		t.Error("Shared() absorbed the cookbook; the handshake would carry it")
	}
}

// TestCLITeachesTheReadSide holds ADR-063: a caller with only the binary is
// told to plan one read of every site, so the document must teach the address
// forms and the flags that find sites it cannot name — and where a plan reads
// an address differently from a read.
func TestCLITeachesTheReadSide(t *testing.T) {
	got := CLI()
	for _, must := range []string{
		"'b.go:/func Start/,+12'",
		"PATH:RANGE[,RANGE...]",
		"N- (to the end)",
		"-M (from the start)",
		"A,+N",
		"$ (the last line)",
		"/from/,/to/",
		"but not -M or a comma list",
		"must match exactly once",
		"also serves, and licenses, the line after each range that ends before the file does (ADR-161)",
		"--grep PATTERN",
		"--exclude GLOB",
		"--ast-grep PATTERN",
		"--files-from FILE",
		"rg -l X . |",
		"A read exits 1 when a range cannot be served",
		"an end past the last line is clamped, not an error",
		"mrw: nothing was recorded", // ADR-166: a read in a read-only sandbox
		"XDG_STATE_HOME",
		"a .git directory the walk meets is skipped, but one you name is walked",
		".gitignore",
	} {
		if !strings.Contains(got, must) {
			t.Errorf("CLI() does not teach %q:\n%s", must, got)
		}
	}
	for _, leak := range []string{"--grep", "--files-from", "exactly once"} {
		if strings.Contains(Shared(), leak) {
			t.Errorf("Shared() absorbed %q; the read section stays on CLI()", leak)
		}
	}
}

// ADR-066 (Codex review of #207). A commit that fails after some files landed
// is reported PARTIALLY APPLIED, so the Shared sentence both surfaces serve
// cannot promise that any failed hunk means nothing was written.
func TestSharedSaysAFailedCommitIsReportedPartial(t *testing.T) {
	got := Shared()
	if strings.Contains(got, "if any hunk fails, nothing is written") {
		t.Errorf("Shared still promises nothing is written whenever a hunk fails:\n%s", got)
	}
	if !strings.Contains(got, "PARTIALLY APPLIED") {
		t.Errorf("Shared does not say a failed commit is reported PARTIALLY APPLIED:\n%s", got)
	}
}

// ADR-066 (Codex review of #207, round 3). Every sentence the binary serves
// about failure is read, not just Shared: a commit failure can leave some files
// written with a hunk failed, so no served text may promise that any failed
// hunk means nothing was written.
func TestNoServedTextPromisesNothingWrittenOnAnyFailure(t *testing.T) {
	for _, text := range []string{Shared(), WhyAllOrNothing(), CLI()} {
		for _, stale := range []string{"if any hunk fails, nothing is written", "A failed hunk writes nothing"} {
			if strings.Contains(text, stale) {
				t.Errorf("served text still says %q:\n%s", stale, text)
			}
		}
	}
}

// ADR-069 T12, from the v1.25.1 field test of the downloaded Windows asset:
// `mrw instructions` said nothing about the padded-path refusal or the --
// escape, so a caller who has only the binary learns the rule at the
// refusal. ADR-063 promised the read side's traps here; this is one.
func TestInstructionsTeachThePaddedPathRule(t *testing.T) {
	got := CLI()
	for _, must := range []string{
		"goes after --",
		"mrw read -- 'x '",
		"--files-from 'list '",
	} {
		if !strings.Contains(got, must) {
			t.Errorf("mrw instructions lacks %q", must)
		}
	}
}

// ADR-071 T5. Two sessions of the v1.25.1 round read "per line" as broken:
// after a write, lines never served were writable. That is ADR-002's design
// (a file mrw just wrote is one it knows wholly, ADR-005 §4), kept by M on
// 2026-09-25; the served text now says so instead of leaving it to be found.
func TestInstructionsSayAWrittenFileIsWhollyKnown(t *testing.T) {
	if got := CLI(); !strings.Contains(got, "A file mrw just wrote is wholly known") {
		t.Error("mrw instructions do not state the write exception to the per-line rule")
	}
}

// ADR-161 Decision 4. The full contract opens with ADR-037's five sentences and
// then the recipes: ten numbered label lines ending in ":" (nine from ADR-161,
// the tenth ADR-167's), each followed by its command indented two spaces, no
// prose between; recipe 1 is the sub write,
// recipe 6 carries ADR-092's caveat on its label, no recipe reaches for perl,
// awk or sed, and the reference below keeps the whole over 900 words.
func TestInstructionsLeadWithTheRecipes(t *testing.T) {
	cli := CLI()
	head := Shared() + "\n\nRecipes:\n"
	if !strings.HasPrefix(cli, head) {
		t.Fatalf("CLI() does not open with Shared() and then Recipes:\n%.400s", cli)
	}
	block, _, _ := strings.Cut(cli[len(head):], "\n\n")
	lines := strings.Split(block, "\n")
	if len(lines) != 20 {
		t.Fatalf("the recipes are %d lines, want 10 label lines and 10 command lines:\n%s", len(lines), block)
	}
	for i := 0; i < 20; i += 2 {
		n := i/2 + 1
		label, command := lines[i], lines[i+1]
		if !strings.HasPrefix(label, strconv.Itoa(n)+". ") || !strings.HasSuffix(label, ":") {
			t.Errorf("recipe %d's label is not `%d. …:`: %q", n, n, label)
		}
		if !strings.HasPrefix(command, "  ") || strings.HasPrefix(command, "   ") || !strings.Contains(command, "mrw ") {
			t.Errorf("recipe %d's command is not an mrw command indented two spaces: %q", n, command)
		}
		for _, tool := range []string{"perl", "awk", "sed "} {
			if strings.Contains(label+command, tool) {
				t.Errorf("recipe %d reaches for %s: %q", n, tool, label+"\n"+command)
			}
		}
		// A command line is commands, not prose: each segment between && and |
		// starts with mrw, or is the stand-in for any command (recipe 9) or the
		// printf that pipes a plan (the Codex review of T3: ", then" and "with the
		// plan" made recipes 4 and 5 exit 2 as printed).
		for _, seg := range regexp.MustCompile(`&&|\|`).Split(command, -1) {
			seg = strings.TrimSpace(seg)
			if !strings.HasPrefix(seg, "mrw ") && seg != "cmd" && !strings.HasPrefix(seg, "printf ") {
				t.Errorf("recipe %d's command has a segment that is not a command: %q", n, seg)
			}
		}
		if strings.Contains(command, ", then") || strings.Contains(command, " with ") {
			t.Errorf("recipe %d's command holds prose: %q", n, command)
		}
	}
	if got := lines[1]; got != `  mrw write --grep '\bOld\b' --sub 'New' .` {
		t.Errorf("recipe 1's command is %q", got)
	}
	if !strings.Contains(lines[10], ThenShCaveat()) {
		t.Errorf("recipe 6 does not carry ADR-092's caveat on its label: %q", lines[10])
	}
	if n := len(strings.Fields(cli)); n <= 900 {
		t.Errorf("the full contract is %d words; the recipes were added, not the reference cut, so it stays over 900", n)
	}
}
