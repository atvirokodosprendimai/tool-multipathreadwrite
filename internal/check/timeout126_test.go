package check

import (
	"context"
	"strings"
	"testing"
	"time"
)

// ADR-126. A check whose deadline had passed before it started was reported
// "timed out after …" with Ran false, which the CLI read as "no check could
// run", exit 2 — the configuration problem — though a write had landed and
// nothing verified it. It is TimedOutBeforeStart now, and StoppedBeforeStart
// says so, as it does for an interrupt before the start (ADR-080).
func TestADeadlineBeforeTheStartIsATimeoutNotACannotStart(t *testing.T) {
	needSh(t)
	root := t.TempDir()
	cfg := Config{Check: "exit 0", declared: true}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	res, err := Run(ctx, root, cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ran || res.ExitCode != -1 || !strings.HasPrefix(res.Skipped, TimedOutBeforeStart) || !StoppedBeforeStart(res) {
		t.Errorf("a deadline before the start was not a timeout before it started: %+v", res)
	}
	stop, cancelNow := context.WithCancel(context.Background())
	cancelNow()
	res, err = Run(stop, root, cfg, nil)
	if err != nil || res.Ran || res.Skipped != Interrupted || !StoppedBeforeStart(res) {
		t.Errorf("a cancel before the start stopped being interrupted: err %v, %+v", err, res)
	}
	if StoppedBeforeStart(Result{Skipped: "could not start: exec: no such file"}) {
		t.Error("a check that could not start was called stopped")
	}
}
