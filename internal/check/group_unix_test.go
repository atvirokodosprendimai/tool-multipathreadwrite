//go:build unix

package check

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// ADR-072 T4. The check's timeout killed only sh, so `sh -c 'make test'` left
// make running after mrw reported "timed out". The check runs in its own
// process group and the timeout kills the group.
func TestATimedOutCheckLeavesNoGrandchild(t *testing.T) {
	root := t.TempDir()
	pidFile := filepath.Join(root, "gc.pid")
	res, err := Run(context.Background(), root,
		Config{Check: `sleep 30 & echo $! > gc.pid; wait`, TimeoutSeconds: 1, declared: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Skipped, "timed out") {
		t.Fatalf("want a timed-out check, got %+v", res)
	}
	b, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for syscall.Kill(pid, 0) == nil {
		if time.Now().After(deadline) {
			_ = syscall.Kill(pid, syscall.SIGKILL)
			t.Fatalf("the check's grandchild %d outlived its timeout", pid)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// ADR-072 T4. An interrupt sent to mrw while a check runs cancels the check.
// The check started, so it RAN and did not pass: exit 3, and the receipt says
// why. Driven here by cancelling the context the signal would cancel.
func TestAnInterruptedCheckSaysSo(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(300*time.Millisecond, cancel)
	start := time.Now()
	res, err := Run(ctx, t.TempDir(), Config{Check: "sleep 30", declared: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Fatalf("an interrupted check took %s to return", d)
	}
	if !res.Ran || res.OK() || res.Skipped != "interrupted" {
		t.Fatalf("want a check that ran, did not pass, and says interrupted: %+v", res)
	}
}

// ADR-072, review of #229 (B1). A hangup ends mrw, and the check, in a process
// group of its own, never hears it, so a hangup stops the check too. A signal
// the process was started with ignored stays ignored: nohup ignores SIGHUP.
func TestTheCheckStopsOnHangupUnlessHangupIsIgnored(t *testing.T) {
	if signal.Ignored(syscall.SIGHUP) {
		t.Skip("this test process was started with SIGHUP ignored")
	}
	has := func(s os.Signal) bool {
		for _, x := range checkSignals() {
			if x == s {
				return true
			}
		}
		return false
	}
	if !has(syscall.SIGHUP) || !has(syscall.SIGTERM) || !has(os.Interrupt) {
		t.Fatalf("a check does not stop on hangup, terminate and interrupt: %v", checkSignals())
	}
	signal.Ignore(syscall.SIGHUP)
	defer signal.Reset(syscall.SIGHUP)
	if has(syscall.SIGHUP) {
		t.Fatal("a hangup the process ignores was switched back on for the check")
	}
}
