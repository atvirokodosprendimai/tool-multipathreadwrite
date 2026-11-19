package main

import (
	"strings"
	"testing"
)

// grep takes a count attached to its flag: `-A3`, `-B2`, `-m5`. urfave reads
// that as one flag named A3, so the refusal spelled it `--A3`, which nobody
// typed, and gave no pointer, though `-A 3` gets one. Counted 2026-11-19 across
// Claude Code transcripts outside this repository (a one-off tally, aggregates
// only): 43 calls were refused for `-AN` in one spelling or the other, beside
// 57 for `-A`. The refusal now spells the flag with one dash and ends in the
// sentence its letter has (ADR-140).
func TestAGrepFlagWithItsCountAttachedIsNamed(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	const context = "context is symmetric: -C N prints N lines either side of a match"
	for arg, hint := range map[string]string{"-A3": context, "-B12": context, "-m5": "--max-matches N"} {
		_, err := runSplit(t, "read", arg, "--grep", "x", "f.txt")
		spelled := "flag provided but not defined: -" + strings.TrimLeft(arg, "-") + " (see: mrw read --help); "
		if err == nil || exitCode(err) != exitUsage || !strings.Contains(err.Error(), spelled) || !strings.Contains(err.Error(), hint) {
			t.Errorf("read %s: %v, want exit %d holding %q and %q", arg, err, exitUsage, spelled, hint)
		}
	}
	// What is not a table letter with a count keeps the refusal it had: a long
	// name gets its second dash back and no sentence.
	for arg, want := range map[string]string{"--A3x": "flag provided but not defined: --A3x (see: mrw read --help)", "-z3": "flag provided but not defined: --z3 (see: mrw read --help)", "--bogus": "flag provided but not defined: --bogus (see: mrw read --help)"} {
		_, err := runSplit(t, "read", arg, "f.txt")
		if err == nil || exitCode(err) != exitUsage || err.Error() != want {
			t.Errorf("read %s: %v, want exactly %q", arg, err, want)
		}
	}
	// Two dashes typed are two dashes reported: urfave says -A3 for both, so
	// the arguments are what tells them apart, and only a built binary has them.
	run, _ := ifChangedRun(t, map[string]string{"f.txt": "one\n"})
	for arg, want := range map[string]string{"--A3": "not defined: --A3 (see: mrw read --help); " + context, "-A3": "not defined: -A3 (see: mrw read --help); " + context} {
		if out, errOut, code := run("", "read", arg, "--grep", "x", "f.txt"); code != exitUsage || out != "" || !strings.Contains(errOut, want) {
			t.Errorf("read %s: exit %d, stdout %q, want exit %d holding %q:\n%s", arg, code, out, exitUsage, want, errOut)
		}
	}
	// A usage error that names no refused flag is worded as before.
	if _, err := runSplit(t, "read", "--max-lines"); err == nil || exitCode(err) != exitUsage || strings.Contains(err.Error(), "not defined") {
		t.Errorf("read --max-lines: %v, want exit %d for the missing value", err, exitUsage)
	}
	// A flag that takes no count is not given one: -i3 is no spelling of -i.
	if _, err := runSplit(t, "read", "-i3", "f.txt"); err == nil || strings.Contains(err.Error(), "(?i)") {
		t.Errorf("read -i3: %v, want no sentence for -i", err)
	}
}
