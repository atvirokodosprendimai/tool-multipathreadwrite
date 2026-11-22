package bench

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanariesReportTheOriginalMalformedTranscriptLine(t *testing.T) {
	for _, profile := range []bool{false, true} {
		transcript := []byte(profileInitA + "\n" + hookLine("SessionStart", "SessionStart:startup") + "\nbroken\n" + profileResult)
		var problems []string
		if profile {
			problems = CheckProfileCanary(transcript, Configs["P"], "", "", "")
		} else {
			problems = CheckCanary(transcript, Configs["A"], "", "")
		}
		if !strings.Contains(strings.Join(problems, "\n"), "transcript line 3") {
			t.Errorf("profile=%v, want original line 3 in %v", profile, problems)
		}
	}
}

func TestAttemptKeepsSpendAndAllEvidenceFailures(t *testing.T) {
	o, _ := runOptions(t)
	o.Claude = filepath.Join(t.TempDir(), "claude")
	put(t, o.Claude, `#!/bin/sh
echo '{"type":"system","subtype":"init"}'
echo 'broken'
echo '{"type":"result","total_cost_usd":0.25,"usage":{"input_tokens":11}}'
rm -f "$TMPDIR/hook.log"
mkdir -p "$XDG_STATE_HOME/mrw/x"
echo x > "$XDG_STATE_HOME/mrw/x/seen"
`, 0755)
	a := attempt{ID: "lost-evidence", Task: o.Tasks[0], Model: "stub-model", Config: Configs["B"], Repeat: 1}
	r := runAttempt(o, a, 0, "stub", "stub")
	if r.CostUSD == nil || *r.CostUSD != 0.25 || r.Usage.InputTokens != 11 || r.Exclusion != ExclusionSetup {
		t.Fatalf("cost, usage, exclusion: %v, %+v, %q (%s)", r.CostUSD, r.Usage, r.Exclusion, r.Note)
	}
	for _, want := range []string{"transcript line 2", "hook log", "slipped writes"} {
		if !strings.Contains(r.Note, want) {
			t.Errorf("note %q omits %q", r.Note, want)
		}
	}
}

func TestHookRecordingFailureReachesItsCaller(t *testing.T) {
	log := t.TempDir() // opening a directory for append fails on every platform
	for _, cmd := range []string{"rm file", "cat file"} {
		var stderr bytes.Buffer
		if code := Hook(bytes.NewReader(hookInput(t, "Bash", cmd)), &stderr, log, ""); code != 2 || !strings.Contains(stderr.String(), "hook log") {
			t.Errorf("hook %q: exit %d, stderr %q", cmd, code, stderr.String())
		}
		if cmd == "rm file" && !strings.Contains(stderr.String(), "refused:") {
			t.Errorf("refusal absent from %q", stderr.String())
		}
	}
}

func TestALaterSuccessfulHookCannotHideARecordingFailure(t *testing.T) {
	if os.Getenv("TEST_BENCH_RECORD_HOOK") == "yes" {
		os.Exit(Hook(os.Stdin, os.Stderr, filepath.Join(os.TempDir(), "hook.log"), ""))
	}
	o, _ := runOptions(t)
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	o.Out = "receipts" // the hook runs in a different working directory
	o.Claude = filepath.Join(t.TempDir(), "claude")
	put(t, o.Claude, `#!/bin/sh
set -eu
case "$MRW_BENCH_HOOK_READY" in */receipts/*) exit 71 ;; esac
log="$TMPDIR/hook.log"
hook() { echo '{"tool_name":"Bash","tool_input":{"command":"cat notes.txt"}}' | TEST_BENCH_RECORD_HOOK=yes `+shellQuote(binary)+` -test.run='^TestALaterSuccessfulHookCannotHideARecordingFailure$'; }
rm "$log"
hook && exit 70
: > "$log"
hook
echo '{"type":"system","subtype":"init"}'
echo '{"type":"result","total_cost_usd":0.25}'
`, 0755)
	a := attempt{ID: "lost-hook", Task: o.Tasks[0], Config: Configs["B"], Repeat: 1}
	r := runAttempt(o, a, 0, "stub", "stub")
	if r.ExitCode != 0 || r.HookReads != 1 || r.Exclusion != ExclusionSetup || !strings.Contains(r.Note, "hook log") {
		t.Fatalf("exit %d, reads %d, exclusion %q, note %q", r.ExitCode, r.HookReads, r.Exclusion, r.Note)
	}
}

func TestABadUsageFieldDoesNotEraseTheReportedSpend(t *testing.T) {
	raw := []byte(`{"type":"result","usage":{"input_tokens":"lost"},"total_cost_usd":0.5}`)
	record, err := parseTranscript(raw)
	if err == nil || record.CostUSD == nil || *record.CostUSD != 0.5 {
		t.Fatalf("cost %v, err %v", record.CostUSD, err)
	}
}
func TestAnOversizedTranscriptLinePreservesTheFollowingCost(t *testing.T) {
	result := `{"type":"result","total_cost_usd":0.5}`
	line := make([]byte, 1<<26+1, 1<<26+2+len(result))
	line = append(line, '\n')
	record, err := parseTranscript(append(line, result...))
	if err == nil || !strings.Contains(err.Error(), "transcript line 1") || record.CostUSD == nil || *record.CostUSD != 0.5 {
		t.Fatalf("cost %v, err %v", record.CostUSD, err)
	}
}

func TestHookCountsRejectPartialRows(t *testing.T) {
	log := filepath.Join(t.TempDir(), "log")
	for _, contents := range []string{"refused\tcmd", "bad row\n"} {
		put(t, log, contents, 0600)
		if _, _, err := hookCounts(log); err == nil {
			t.Errorf("accepted incomplete hook evidence %q", contents)
		}
	}
}
