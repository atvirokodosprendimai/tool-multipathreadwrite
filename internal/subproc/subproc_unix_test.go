//go:build unix

package subproc

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// readPid waits for a child to write its pid, then returns it.
func readPid(t *testing.T, path string) int {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		b, err := os.ReadFile(path)
		if err == nil {
			if pid, err := strconv.Atoi(strings.TrimSpace(string(b))); err == nil && pid > 0 {
				return pid
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("no pid in %s", path)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// waitGone polls until pid no longer exists, killing it and failing if it
// outlives the bound, so a red run leaves nothing behind.
func waitGone(t *testing.T, pid int, why string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for syscall.Kill(pid, 0) == nil {
		if time.Now().After(deadline) {
			_ = syscall.Kill(pid, syscall.SIGKILL)
			t.Fatalf("%s: process %d is still running", why, pid)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// ADR-072 T4. A check's deadline killed only the process mrw started: sh -c
// 'make test' left make running. The child is its own process group, and the
// group is what the deadline kills.
func TestATimedOutCommandTakesItsGrandchildWithIt(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "pid")
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	c := Command(ctx, "sh", "-c", `sleep 30 & echo $! > "$1"; wait`, "sh", pidFile)
	start := time.Now()
	_ = c.Run()
	if d := time.Since(start); d > 5*time.Second {
		t.Fatalf("Run took %s after a 500ms deadline", d)
	}
	waitGone(t, readPid(t, pidFile), "the grandchild outlived its parent's deadline")
}

// ADR-072 T4 (and ADR-074 T1, which puts --ast-grep on this). A grandchild
// that holds stdout kept Output waiting past the deadline: 30 s behind a 2 s
// bound in the round. This one leaves the group, so the group kill cannot
// reach it and only the bound on the wait for its pipe can end Output.
func TestAHeldPipeIsNotWaitedOnForever(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "pid")
	script := filepath.Join(dir, "hold.sh")
	body := "perl -e 'setpgrp(0,0); open(my $f, \">\", $ARGV[0]) or die; print $f $$; close $f; sleep 30' \"$1\" &\nwait\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	c := Command(ctx, "sh", script, pidFile)
	start := time.Now()
	_, _ = c.Output()
	d := time.Since(start)
	_ = syscall.Kill(readPid(t, pidFile), syscall.SIGKILL)
	if d > 5*time.Second {
		t.Fatalf("Output waited %s for a pipe a grandchild held", d)
	}
}
