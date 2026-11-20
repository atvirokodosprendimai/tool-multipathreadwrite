package main

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/bench"
)

// TestTheRunPinFlagsReachTheirOptions: ADR-206 T1's flags go through the
// production entry point, bench run's runWith, to the options a stage starts
// with; --mrw-sha256 and --mrw-instructions are shared by run and canary, and
// --keep-state-file, --keep-diff and --max-spend are run's alone. A spend stop
// ends bench with status 3, and the disk floor and a start-error stop with 2.
func TestTheRunPinFlagsReachTheirOptions(t *testing.T) {
	sha := strings.Repeat("ab", 32)
	tasks := filepath.Join("..", "..", "bench", "tasks")
	all, err := bench.LoadTasks(tasks)
	if err != nil {
		t.Fatal(err)
	}
	measured, err := bench.SelectTasks(all, nil)
	if err != nil || len(measured) == 0 {
		t.Fatalf("the corpus has no measured task: %v", err)
	}
	var got bench.RunOptions
	args := []string{"--stage", "s", "--out", t.TempDir(), "--model", "m", "--claude", "/c", "--mrw", "/m", "--config-dir", "/d",
		"--containment", "bypass", "--canary", "/k", "--tasks-dir", tasks, "--task", measured[0].ID,
		"--mrw-sha256", sha, "--mrw-instructions", "i.txt", "--keep-state-file", "mrw/x.jsonl", "--keep-diff", "--max-spend", "12.5"}
	if err := runWith(args, func(o bench.RunOptions) error { got = o; return nil }); err != nil {
		t.Fatal(err)
	}
	if got.MrwSHA256 != sha || got.MrwInstructions != "i.txt" || got.KeepStateFile != "mrw/x.jsonl" || !got.KeepDiff || got.MaxSpend != 12.5 {
		t.Errorf("run's flags reached %+v", got)
	}
	cfs, co, _, _ := stageFlags("canary")
	if err := cfs.Parse([]string{"--mrw-sha256", sha, "--mrw-instructions", "i.txt"}); err != nil {
		t.Fatal(err)
	}
	if co.MrwSHA256 != sha || co.MrwInstructions != "i.txt" {
		t.Errorf("canary's flags reached %+v", *co)
	}
	for _, f := range []string{"--keep-state-file=x", "--keep-diff", "--keep-scratch-file=x"} {
		if err := canary([]string{f}); err == nil || !strings.Contains(err.Error(), strings.TrimPrefix(strings.SplitN(f, "=", 2)[0], "--")) {
			t.Errorf("bench canary %s: %v, want it refused as a flag only bench run has", f, err)
		}
	}
	for _, c := range []struct {
		err  error
		want int
	}{
		{fmt.Errorf("run: %w", bench.ErrSpendStop), 3},
		{fmt.Errorf("run: %w", bench.ErrDiskFloor), 2},
		{fmt.Errorf("run: %w", bench.ErrStartErrors), 2},
		{fmt.Errorf("run: something else"), 1},
	} {
		var out bytes.Buffer
		if code := report(c.err, &out); code != c.want || !strings.Contains(out.String(), c.err.Error()) {
			t.Errorf("bench ends %q with status %d, printing %q; want %d", c.err, code, out.String(), c.want)
		}
	}
}

// ADR-211 Decision 9: bench canary takes --max-spend, the flag reaches the canary's
// options, and a spend stop ends bench canary with status 3.
func TestBenchCanaryTakesMaxSpend(t *testing.T) {
	tasks := filepath.Join("..", "..", "bench", "tasks")
	var got bench.RunOptions
	stop := fmt.Errorf("canary: %w", bench.ErrSpendStop)
	args := []string{"--out", t.TempDir(), "--model", "m", "--claude", "/c", "--mrw", "/m", "--config-dir", "/d", "--containment", "bypass",
		"--tasks-dir", tasks, "--max-spend", "0.75"}
	err := canaryWith(args, func(o bench.RunOptions, _ bench.Task, _ string) (bench.CanaryReceipt, error) {
		got = o
		return bench.CanaryReceipt{}, stop
	})
	if got.MaxSpend != 0.75 {
		t.Errorf("--max-spend reached the canary's options as %v, want 0.75", got.MaxSpend)
	}
	var out bytes.Buffer
	if code := report(err, &out); !errors.Is(err, bench.ErrSpendStop) || code != 3 || !strings.Contains(out.String(), stop.Error()) {
		t.Errorf("a canary stopped on its spend ends bench with status %d printing %q, want 3", code, out.String())
	}
}
