package check

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// depthOf is what a command wrote to name under root, trimmed.
func depthOf(t *testing.T, root, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		t.Fatalf("%s was not written: %v", name, err)
	}
	return strings.TrimSpace(string(b))
}

// ADR-095 T1, the record's Enforced-by. Every project command mrw starts —
// the check and each step — runs one level deeper than mrw itself: the check
// inherited its caller's value unchanged, so a check that ran mrw was outside
// the depth guard.
func TestTheCheckAndEveryStepRunOneLevelDeeper(t *testing.T) {
	needShell(t)
	tempDirForLogs(t)
	root := t.TempDir()
	t.Setenv("MRW_STEP_DEPTH", "3")
	res, err := Run(context.Background(), root, Config{Check: "printf '%s' \"$MRW_STEP_DEPTH\" > checkdepth"}, nil)
	if err != nil || !res.OK() {
		t.Fatalf("the check did not pass: %v %+v", err, res)
	}
	if got := depthOf(t, root, "checkdepth"); got != "4" {
		t.Errorf("the check saw MRW_STEP_DEPTH=%q under 3, want 4", got)
	}
	steps := RunSteps(context.Background(), root, Config{}, []Step{{Name: "d", Command: "printf '%s' \"$MRW_STEP_DEPTH\" > stepdepth"}})
	if len(steps.Steps) != 1 || steps.Steps[0].Status != StepPass {
		t.Fatalf("the step did not pass: %+v", steps.Steps)
	}
	if got := depthOf(t, root, "stepdepth"); got != "4" {
		t.Errorf("the step saw MRW_STEP_DEPTH=%q under 3, want 4", got)
	}
}

// ADR-095 T1, Decision 3. A value mrw did not write reads as zero and is not
// refused: a check under it sees 1. A readable 8 reads 8 and is the limit.
func TestAnUnreadableDepthCountsAsZero(t *testing.T) {
	needShell(t)
	tempDirForLogs(t)
	for _, v := range []string{"", "x", "-3", " 8", "99999999999999999999"} {
		t.Setenv("MRW_STEP_DEPTH", v)
		if got := StepDepth(); got != 0 {
			t.Errorf("MRW_STEP_DEPTH=%q reads %d, want 0", v, got)
		}
		if err := DepthRefusal("a check is"); err != nil {
			t.Errorf("MRW_STEP_DEPTH=%q is refused: %v", v, err)
		}
		root := t.TempDir()
		res, err := Run(context.Background(), root, Config{Check: "printf '%s' \"$MRW_STEP_DEPTH\" > checkdepth"}, nil)
		if err != nil || !res.OK() {
			t.Fatalf("MRW_STEP_DEPTH=%q: the check did not pass: %v %+v", v, err, res)
		}
		if got := depthOf(t, root, "checkdepth"); got != "1" {
			t.Errorf("MRW_STEP_DEPTH=%q: the check saw %q, want 1", v, got)
		}
	}
	t.Setenv("MRW_STEP_DEPTH", "8")
	if got := StepDepth(); got != 8 {
		t.Errorf("MRW_STEP_DEPTH=8 reads %d", got)
	}
	err := DepthRefusal("a check is")
	if err == nil || !strings.Contains(err.Error(), "a check is refused 8 deep (MRW_STEP_DEPTH=8)") {
		t.Errorf("at 8 the refusal is %v", err)
	}
}
