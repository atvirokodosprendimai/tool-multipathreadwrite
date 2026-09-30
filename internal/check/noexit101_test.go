package check

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
)

// ADR-101. A Result whose check produced no exit status carries ExitCode -1,
// never the zero value: a receipt said "exit_code": 0 beside "ran": false, and
// a consumer reading exit_code alone read a pass. Three returns came before any
// status was set — no command, a refused scope, a log that could not be made.
// The pair: a check that ran reports its own status.
func TestARunWithNoProcessHasExitCodeMinusOne(t *testing.T) {
	needSh(t)
	root := t.TempDir() // no harness and no go.mod: no command
	ctx := context.Background()
	if res, err := Run(ctx, root, Config{Check: "true"}, nil); err != nil || !res.Ran || res.ExitCode != 0 {
		t.Fatalf("the pair, a check that ran: err %v, ran %v, exit_code %d; want nil, true, 0", err, res.Ran, res.ExitCode)
	}
	if res, err := Run(ctx, root, Config{}, nil); err != nil || res.Ran || res.ExitCode != -1 {
		t.Errorf("no command: err %v, ran %v, exit_code %d; want nil, false, -1", err, res.Ran, res.ExitCode)
	}
	if res, err := Run(ctx, root, Config{Check: "true"}, []string{"../outside"}); err == nil || res.Ran || res.ExitCode != -1 {
		t.Errorf("a refused scope: err %v, ran %v, exit_code %d; want a refusal, false, -1", err, res.Ran, res.ExitCode)
	}
	gone := filepath.Join(t.TempDir(), "gone")
	t.Setenv("TMPDIR", gone)
	if runtime.GOOS == "windows" {
		t.Setenv("TMP", gone)
		t.Setenv("TEMP", gone)
	}
	if res, err := Run(ctx, root, Config{Check: "true"}, nil); err == nil || res.Ran || res.ExitCode != -1 {
		t.Errorf("a log that cannot be created: err %v, ran %v, exit_code %d; want an error, false, -1", err, res.Ran, res.ExitCode)
	}
	// A step is run by run too: RunSteps copied run's 0 for a step whose log
	// could not be created (the review of #290).
	if s := RunSteps(ctx, root, Config{}, []Step{{Command: "true"}}).Steps[0]; s.Ran || s.ExitCode != -1 {
		t.Errorf("a step whose log cannot be created: ran %v, exit_code %d, status %s; want false, -1", s.Ran, s.ExitCode, s.Status)
	}
}
