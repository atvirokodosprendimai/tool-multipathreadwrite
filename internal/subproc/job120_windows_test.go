//go:build windows

package subproc

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
)

// alive reports whether pid names a running process.
func alive(pid int) bool {
	h, err := syscall.OpenProcess(0x1000, false, uint32(pid)) // PROCESS_QUERY_LIMITED_INFORMATION
	if err != nil {
		return false
	}
	defer func() { _ = syscall.CloseHandle(h) }()
	var code uint32
	if err := syscall.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	return code == 259 // STILL_ACTIVE
}

// grandchildOf waits for the pid the helper's grandchild writes.
func grandchildOf(t *testing.T, pidfile string) int {
	t.Helper()
	for i := 0; i < 200; i++ {
		if b, err := os.ReadFile(pidfile); err == nil && len(b) > 0 {
			pid, err := strconv.Atoi(string(b))
			if err != nil {
				t.Fatal(err)
			}
			return pid
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("the grandchild never wrote its pid")
	return 0
}

// gone waits up to 5 s for pid to stop, then kills it so no test leaks it.
func gone(t *testing.T, pid int) {
	t.Helper()
	for i := 0; i < 100; i++ {
		if !alive(pid) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	if p, err := os.FindProcess(pid); err == nil {
		_ = p.Kill()
	}
	t.Fatalf("grandchild %d outlived the call", pid)
}

// ADR-120. A cancelled check killed only the sh.exe mrw started; its
// grandchildren ran on and held the log open (a Windows peer, 2026-10-02).
func TestACancelledCommandStopsItsGrandchildOnWindows(t *testing.T) {
	pidfile := filepath.Join(t.TempDir(), "pid")
	ctx, cancel := context.WithCancel(context.Background())
	c := Command(ctx, os.Args[0], "-test.run=^TestJobHelper$")
	c.Env = append(os.Environ(), "MRW_JOB_HELPER=spawn-sleep", "MRW_JOB_PIDFILE="+pidfile)
	done := make(chan error, 1)
	go func() { done <- Run(c) }()
	pid := grandchildOf(t, pidfile)
	cancel()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return after the cancel")
	}
	gone(t, pid)
}

// A child that exits 0 leaves no grandchild running: the job is terminated and
// closed when the child exits (ADR-080's reap, on Windows).
func TestACleanExitReapsTheGrandchildOnWindows(t *testing.T) {
	pidfile := filepath.Join(t.TempDir(), "pid")
	c := Command(context.Background(), os.Args[0], "-test.run=^TestJobHelper$")
	c.Env = append(os.Environ(), "MRW_JOB_HELPER=spawn-exit", "MRW_JOB_PIDFILE="+pidfile)
	if err := Run(c); err != nil {
		t.Fatal(err)
	}
	gone(t, grandchildOf(t, pidfile))
}
