package check

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ADR-177 T1. The exported entry runs a command through the check's own runner
// with a timeout and a log of the caller's choosing, and keeps the log of a
// run that passed. Its timeout is the caller's, not the check's five minutes,
// and mrw run asks for RunCeiling, the runner's 24 hours.
func TestTheRunEntryTakesItsOwnTimeout(t *testing.T) {
	if _, _, ok := Shell(); !ok {
		t.Skip("no POSIX shell")
	}
	if RunCeiling != 24*time.Hour {
		t.Errorf("RunCeiling is %s, want the runner's 24-hour ceiling", RunCeiling)
	}
	dir := t.TempDir()
	start := time.Now()
	res, err := Exec(context.Background(), t.TempDir(), "sleep 5", RunSpec{LogDir: dir, LogName: "a.running.log", Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ran || res.Skipped != "timed out after 1s" {
		t.Errorf("a 1 s timeout over sleep 5: ran %v, skipped %q; want ran and %q", res.Ran, res.Skipped, "timed out after 1s")
	}
	if took := time.Since(start); took > 4*time.Second {
		t.Errorf("the run took %s, the entry's 1 s timeout did not stop it", took)
	}
	if res.OutputFile != filepath.Join(dir, "a.running.log") {
		t.Errorf("the log is %q, want the one the caller named", res.OutputFile)
	}

	// The pair: a command that passes quickly keeps its log, which the check
	// would have deleted, and sees one level more depth.
	res, err = Exec(context.Background(), t.TempDir(), "echo hi; echo depth $MRW_STEP_DEPTH", RunSpec{LogDir: dir, LogName: "b.running.log"})
	if err != nil || !res.OK() {
		t.Fatalf("echo: %+v, %v", res, err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "b.running.log"))
	if err != nil || string(b) != fmt.Sprintf("hi\ndepth %d\n", StepDepth()+1) {
		t.Errorf("the kept log is %q, %v; want the output and a depth one higher than mrw's own", b, err)
	}
	if res.Signal() != 0 {
		t.Errorf("a command that exited has signal %d", res.Signal())
	}
}

// A command killed by a signal reports it beside the result, since its exit
// code is -1 (ADR-101) and mrw run exits 128 plus the number.
func TestTheRunEntryReportsTheSignalThatKilledTheCommand(t *testing.T) {
	if _, _, ok := Shell(); !ok || os.PathSeparator == '\\' {
		t.Skip("needs a POSIX shell and unix signals")
	}
	res, err := Exec(context.Background(), t.TempDir(), "kill -TERM $$", RunSpec{LogDir: t.TempDir(), LogName: "k.running.log"})
	if err != nil || !res.Ran || res.Signal() != 15 || res.ExitCode != -1 {
		t.Errorf("kill -TERM $$: %+v, %v; want ran, signal 15, exit_code -1", res, err)
	}
	res, err = Exec(context.Background(), t.TempDir(), "exit 7", RunSpec{LogDir: t.TempDir(), LogName: "e.running.log"})
	if err != nil || res.Signal() != 0 || res.ExitCode != 7 {
		t.Errorf("exit 7: %+v, %v; want signal 0, exit 7", res, err)
	}
}

// ADR-177 Decision 4. Completed run logs older than LogRetention go, then all
// but the newest 64 by name; a running log goes only past 25 hours; a file
// that is not a run log is never touched.
func TestRunLogsArePrunedByAgeAndCount(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	touch := func(name string, age time.Duration) {
		t.Helper()
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte("x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, now.Add(-age), now.Add(-age)); err != nil {
			t.Fatal(err)
		}
	}
	name := func(i int) string { return fmt.Sprintf("run-20261001T0000%02d.000Z-1.log", i) }
	for i := 0; i < 70; i++ {
		age := time.Hour + time.Duration(i)*time.Minute // newer names are OLDER files: only the name decides
		if i < 3 {
			age = LogRetention + time.Hour
		}
		touch(name(i), age)
	}
	touch("run-20261001T000099.000Z-2.running.log", time.Minute)
	touch("run-20261001T000098.000Z-3.running.log", 26*time.Hour)
	touch("notes.txt", 30*24*time.Hour)
	touch("mrw-check-1.log", 30*24*time.Hour)

	if got := PruneRunLogs(dir, now, ""); got != 3+3+1 {
		t.Errorf("pruned %d, want 7: 3 older than %s, 3 beyond the newest 64, 1 running log past 25 hours", got, LogRetention)
	}
	for i := 0; i < 70; i++ {
		_, err := os.Stat(filepath.Join(dir, name(i)))
		if gone := os.IsNotExist(err); gone != (i < 6) {
			t.Errorf("%s: removed %v, want %v", name(i), gone, i < 6)
		}
	}
	for n, want := range map[string]bool{
		"run-20261001T000099.000Z-2.running.log": true,
		"run-20261001T000098.000Z-3.running.log": false,
		"notes.txt":                              true,
		"mrw-check-1.log":                        true,
	} {
		_, err := os.Stat(filepath.Join(dir, n))
		if kept := err == nil; kept != want {
			t.Errorf("%s: kept %v, want %v", n, kept, want)
		}
	}
	if got := PruneRunLogs(dir, now, ""); got != 0 {
		t.Errorf("a second pass pruned %d, want 0", got)
	}
	if got := PruneRunLogs(filepath.Join(dir, "absent"), now, ""); got != 0 {
		t.Errorf("a missing directory pruned %d", got)
	}
}

// A long run's own log must survive its prune: 64 runs that started after it
// and finished before it leave it the oldest name, and the receipt names it.
func TestARunKeepsItsOwnLogWhenSixtyFourOthersFinishedBeforeIt(t *testing.T) {
	dir := t.TempDir()
	t0 := time.Now().Add(-30 * time.Minute)
	for i := 1; i <= RunLogsKept; i++ {
		_, done := RunLogNames(t0.Add(time.Duration(i)*time.Second), 1000+i)
		if err := os.WriteFile(filepath.Join(dir, done), []byte("x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	_, mine := RunLogNames(t0, 42)
	if err := os.WriteFile(filepath.Join(dir, mine), []byte("the long run's output\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	n := PruneRunLogs(dir, time.Now(), mine)
	if _, err := os.Stat(filepath.Join(dir, mine)); err != nil {
		t.Fatalf("pruned %d; the run's own log is gone: %v", n, err)
	}
	if n != 0 {
		t.Errorf("pruned %d; the 64 others are the newest 64 and stay", n)
	}
	// The pair: the same directory without the exemption loses that log.
	if n := PruneRunLogs(dir, time.Now(), ""); n != 1 {
		t.Errorf("without the exemption %d were pruned, want the one oldest name", n)
	}
}

// The entry's zero Timeout is the 24-hour ceiling, not the check's five
// minutes and not no deadline: an outer 1 s context stops sleep 5 first, and
// the report says which deadline it was.
func TestTheRunEntryWithNoTimeoutAsksForTheCeiling(t *testing.T) {
	if _, _, ok := Shell(); !ok {
		t.Skip("no POSIX shell")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	res, err := Exec(ctx, t.TempDir(), "sleep 5", RunSpec{LogDir: t.TempDir(), LogName: "o.running.log"})
	if err != nil || res.Skipped != "timed out after 24h0m0s" {
		t.Errorf("skipped %q, %v; want the 24-hour ceiling named", res.Skipped, err)
	}
}
