//go:build unix

package check

import (
	"context"
	"strings"
	"testing"
)

// ADR-095 T2. A check or a step whose shell exits 0 from a TERM trap was
// stopped, not passed: after a cancel the verdict comes from the context,
// never the exit status (ADR-003). Nothing exited 0 from the SIGKILL mrw sent
// before; the TERM it sends first makes this reachable.
func TestACheckOrStepThatExitsZeroOnTermIsNotAPass(t *testing.T) {
	tempDirForLogs(t)
	trap := "trap 'exit 0' TERM; sleep 30 & wait"
	res, err := Run(context.Background(), t.TempDir(), Config{Check: trap, TimeoutSeconds: 1, declared: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.OK() || !strings.Contains(res.Skipped, "timed out") {
		t.Errorf("a check that exits 0 on TERM: %+v, want timed out and not OK", res)
	}
	steps := RunSteps(context.Background(), t.TempDir(), Config{TimeoutSeconds: 1}, []Step{{Name: "t", Command: trap}})
	if len(steps.Steps) != 1 || steps.Steps[0].Status != StepTimedOut {
		t.Errorf("a step that exits 0 on TERM: %+v, want timed_out", steps.Steps)
	}
}
