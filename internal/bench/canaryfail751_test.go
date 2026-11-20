package bench

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// Review of #751. An attempt that records no cost is the missing-cost stop, which
// ADR-211 Decision 9 says ends the canary with exit 3 and writes canary.json. A
// transcript that cannot be read is no reason to return before the stop: the
// attempt's missing cost is already known.
func TestACanaryAttemptWithNoCostStopsBeforeItsTranscriptIsRead(t *testing.T) {
	rig := newPinRig(t, []string{"A", "B"}, 1)
	rig.costs(t, "none", "0.10")
	co := rig.canaryOpts()
	co.MaxSpend = 0.50
	// The transcript's path is a directory: the attempt cannot write it, and the canary cannot read it.
	if err := os.MkdirAll(filepath.Join(outDir(co, Configs["A"]), "canary.A.transcript.jsonl"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := RunCanary(co, co.Tasks[0], "")
	if !errors.Is(err, ErrSpendStop) || ExitCode(err) != 3 {
		t.Fatalf("%v, want ErrSpendStop and exit 3, not the transcript's read error", err)
	}
	if m := canaryFile(t, rig); got.Pass || m["pass"] != false || len(got.Problems["spend"]) == 0 {
		t.Errorf("pass %v, canary.json %v, problems %v, want it written pass false under spend", got.Pass, m, got.Problems)
	}
}
