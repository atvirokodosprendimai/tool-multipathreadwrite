package main

import (
	"strings"
	"testing"
)

// TestTheRunPinFlagsReachTheirOptions: ADR-206 T1's flags are parsed by the
// real flag sets. --mrw-sha256 and --mrw-instructions are shared by run and
// canary; --keep-state-file, --keep-diff and --max-spend are run's alone.
func TestTheRunPinFlagsReachTheirOptions(t *testing.T) {
	sha := strings.Repeat("ab", 32)
	fs, o, _, _ := stageFlags("run")
	runPinFlags(fs, o)
	args := []string{"--mrw-sha256", sha, "--mrw-instructions", "i.txt", "--keep-state-file", "mrw/x.jsonl", "--keep-diff", "--max-spend", "12.5"}
	if err := fs.Parse(args); err != nil {
		t.Fatal(err)
	}
	if o.MrwSHA256 != sha || o.MrwInstructions != "i.txt" || o.KeepStateFile != "mrw/x.jsonl" || !o.KeepDiff || o.MaxSpend != 12.5 {
		t.Errorf("run's flags reached %+v", *o)
	}
	cfs, co, _, _ := stageFlags("canary")
	if err := cfs.Parse([]string{"--mrw-sha256", sha, "--mrw-instructions", "i.txt"}); err != nil {
		t.Fatal(err)
	}
	if co.MrwSHA256 != sha || co.MrwInstructions != "i.txt" {
		t.Errorf("canary's flags reached %+v", *co)
	}
	for _, f := range []string{"--keep-state-file=x", "--keep-diff", "--max-spend=1"} {
		cfs, _, _, _ := stageFlags("canary")
		if err := cfs.Parse([]string{f}); err == nil {
			t.Errorf("bench canary took %s, which only bench run has", f)
		}
	}
}
