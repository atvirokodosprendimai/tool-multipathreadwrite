package bench

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// qhRedCommit is the quality-harness commit every qh task is pinned to — the
// seven of Stage 1 and the three ADR-158 added — at which its declared check
// was measured red (ADR-156 T1 S6, 2026-10-14).
const qhRedCommit = "bbf82ff3c32c95c4251ac2eee8d0e6cfd8bad999"

// TestTheTasksWhoseCheckIsRedAtBaseSayWhy: every task pinned to the measured
// quality-harness commit carries check_red_at_base with a reason, and no other
// task does — the flag is a measured fact, not a label to spread.
func TestTheTasksWhoseCheckIsRedAtBaseSayWhy(t *testing.T) {
	ts, err := LoadTasks(tasksDir(t))
	if err != nil {
		t.Fatal(err)
	}
	red := 0
	for _, task := range ts {
		measured := task.Repo == "quality-harness" && task.Commit == qhRedCommit
		says := strings.TrimSpace(task.CheckRedAtBase) != ""
		if measured != says {
			t.Errorf("%s: check_red_at_base %q, want a reason exactly when pinned to quality-harness@%s", task.ID, task.CheckRedAtBase, qhRedCommit[:8])
		}
		if says {
			red++
		}
	}
	if red != 10 {
		t.Errorf("tasks flagged red at base: %d, want the 10 quality-harness tasks", red)
	}
}

// TestTheAttemptPathLinksTheToolchainATaskCheckNeeds: the attempt's PATH links
// node, npm, npx and python3 beside git, go and gofmt, each to the host's own
// file, and never an mrw that sits in the same directory.
func TestTheAttemptPathLinksTheToolchainATaskCheckNeeds(t *testing.T) {
	// exec.LookPath on Windows finds only names with a PATHEXT extension, and
	// toolPath links by the bare name, so these stubs are not found there.
	if runtime.GOOS == "windows" {
		t.Skip("bench runs on the operator's macOS or Linux machine; Windows is out of ADR-156's scope")
	}
	host := t.TempDir()
	for _, n := range []string{"git", "go", "gofmt", "node", "npm", "npx", "python3", "mrw"} {
		if err := os.WriteFile(filepath.Join(host, n), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", host)
	dir, err := toolPath(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"git", "go", "gofmt", "node", "npm", "npx", "python3"} {
		got, err := os.Readlink(filepath.Join(dir, n))
		if err != nil {
			t.Errorf("%s: not linked: %v", n, err)
			continue
		}
		want, err := filepath.EvalSymlinks(filepath.Join(host, n))
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("%s links %s, want %s", n, got, want)
		}
	}
	if _, err := os.Lstat(filepath.Join(dir, "mrw")); !os.IsNotExist(err) {
		t.Errorf("mrw reached the attempt's PATH: %v", err)
	}
}

// TestAReceiptCarriesItsTasksRedCheck: an attempt of a task flagged red at base
// says so in its receipt, and an attempt of an unflagged task does not.
func TestAReceiptCarriesItsTasksRedCheck(t *testing.T) {
	for _, flagged := range []bool{true, false} {
		opts, _ := runOptions(t)
		if flagged {
			opts.Tasks[0].CheckRedAtBase = "measured red at base"
		}
		if err := Run(opts); err != nil {
			t.Fatal(err)
		}
		got := receipts(t, opts.Out)
		if len(got) == 0 {
			t.Fatal("no receipts")
		}
		for name, b := range got {
			var r Receipt
			if err := json.Unmarshal(b, &r); err != nil {
				t.Fatal(err)
			}
			if r.CheckRedAtBase != flagged {
				t.Errorf("%s: check_red_at_base %v, want %v", name, r.CheckRedAtBase, flagged)
			}
		}
	}
}

// TestADescriptiveRowLeavesOutTasksWhoseCheckIsRedAtBase: beside the verdict,
// a group holding flagged attempts gets one row computed without them, marked
// descriptive; the verdict row still counts every task, and a group with no
// flagged attempt gets no second row.
func TestADescriptiveRowLeavesOutTasksWhoseCheckIsRedAtBase(t *testing.T) {
	rs := receiptsFor("sonnet", "own", repeat(1.0, 10), repeat(0.5, 10), repeat(true, 10), repeat(true, 10))
	red := receiptsFor("sonnet", "own", repeat(1.0, 4), repeat(4.0, 4), repeat(true, 4), repeat(true, 4))
	for i := range red {
		red[i].Task = "red-" + red[i].Task
		red[i].CheckRedAtBase = true
	}
	rs = append(rs, red...)
	rs = append(rs, receiptsFor("opus", "own", repeat(1.0, 10), repeat(0.5, 10), repeat(true, 10), repeat(true, 10))...)
	got, err := Analyze(rs)
	if err != nil {
		t.Fatal(err)
	}
	var verdict, descriptive, opus int
	for _, r := range got {
		switch {
		case r.Model == "opus":
			opus++
			if r.Descriptive != "" {
				t.Errorf("opus has no flagged attempt but got a descriptive row: %+v", r)
			}
		case r.Descriptive == "":
			verdict++
			if r.Tasks != 14 {
				t.Errorf("verdict row: %d tasks, want every task (14)", r.Tasks)
			}
		default:
			descriptive++
			if r.Tasks != 10 || r.Attempts != 20 || r.Verdict != "PAYS" {
				t.Errorf("descriptive row: %+v, want the 10 unflagged tasks only, PAYS", r)
			}
		}
	}
	if verdict != 1 || descriptive != 1 || opus != 1 {
		t.Errorf("rows: %d verdict, %d descriptive, %d opus; want 1, 1, 1", verdict, descriptive, opus)
	}
}
