//go:build unix

package subproc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"
)

// script writes body to a file of its own in a fresh directory and returns
// its path, so a shell script needs no quoting inside a Go string.
func script(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "s.sh")
	if err := os.WriteFile(p, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

// waitFile waits up to three seconds for path to exist.
func waitFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if _, err := os.Stat(path); err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s never appeared", path)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// trapper traps TERM to write "$1", writes its pid to "$2", and waits.
const trapper = "trap 'echo term > \"$1\"; exit 0' TERM\necho $$ > \"$2\"\nsleep 30 &\nwait\n"

// ADR-095 T2. A cancel sends the group SIGTERM before SIGKILL, so a nested mrw
// can stop its own check: a shell that traps TERM writes its marker before Run
// returns. With SIGKILL at once no trap ran.
func TestACancelSendsTermBeforeKill(t *testing.T) {
	dir := t.TempDir()
	marker, pid := filepath.Join(dir, "marker"), filepath.Join(dir, "pid")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := Command(ctx, "sh", script(t, trapper), marker, pid)
	done := make(chan error, 1)
	go func() { done <- Run(c) }()
	p := readPid(t, pid)
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = syscall.Kill(p, syscall.SIGKILL)
		t.Fatal("Run did not return after the cancel")
	}
	if _, err := os.Stat(marker); err != nil {
		t.Errorf("the TERM trap never ran: %v", err)
	}
}

// ADR-095 T2. The cancel returns only once the group is empty: the leader has
// no trap and dies of TERM at once, while a member traps TERM, takes 0.3 s and
// writes its marker. The marker is there the moment the cancel returns.
func TestACancelWaitsForASlowMemberToLeave(t *testing.T) {
	dir := t.TempDir()
	marker, ready := filepath.Join(dir, "marker"), filepath.Join(dir, "ready")
	member := script(t, "trap 'sleep 0.3; echo left > \"$1\"; exit 0' TERM\ntouch \"$2\"\nsleep 30 &\nwait\n")
	c := Command(context.Background(), "sh", "-c", "sh \"$1\" \"$2\" \"$3\" & wait", "sh", member, marker, ready)
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	waited := make(chan error, 1)
	go func() { waited <- c.Wait() }()
	waitFile(t, ready)
	_ = c.Cancel()
	_, err := os.Stat(marker)
	<-waited
	if err != nil {
		t.Errorf("the cancel returned before the slow member left: %v", err)
	}
}

// ADR-095 T2. A group whose members ignore TERM is killed once the grace has
// passed: nothing is left when the cancel returns.
func TestACancelKillsWhatIgnoresTerm(t *testing.T) {
	pid := filepath.Join(t.TempDir(), "pid")
	c := Command(context.Background(), "sh", "-c", "trap '' TERM; sleep 30 & echo $! > \"$1\"; wait", "sh", pid)
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	waited := make(chan error, 1)
	go func() { waited <- c.Wait() }()
	p := readPid(t, pid)
	_ = c.Cancel()
	waitGone(t, p, "a member that ignores TERM outlived the cancel")
	<-waited
}

// ADR-095 T2. The reap after a clean exit sends TERM first: a straggler the
// leader left, which traps TERM, writes its marker and is gone when Run
// returns.
func TestAStragglerAfterACleanExitGetsTermFirst(t *testing.T) {
	dir := t.TempDir()
	marker, pid := filepath.Join(dir, "marker"), filepath.Join(dir, "pid")
	member := script(t, trapper)
	leader := "sh \"$1\" \"$2\" \"$3\" >/dev/null 2>&1 &\nwhile [ ! -s \"$3\" ]; do sleep 0.02; done\nexit 0\n"
	if err := Run(Command(context.Background(), "sh", "-c", leader, "sh", member, marker, pid)); err != nil {
		t.Fatal(err)
	}
	p := readPid(t, pid)
	if _, err := os.Stat(marker); err != nil {
		t.Errorf("the straggler never heard TERM: %v", err)
	}
	waitGone(t, p, "the straggler outlived the reap")
}

// call is one signal subproc sent a group, and the answer it got.
type call struct {
	pid int
	sig syscall.Signal
	err error
}

// recordSignals makes signalGroup answer through answer and record each call,
// restoring it when the test ends.
func recordSignals(t *testing.T, answer func(pid int, sig syscall.Signal) error) func() []call {
	t.Helper()
	var mu sync.Mutex
	var calls []call
	old := signalGroup
	signalGroup = func(pid int, sig syscall.Signal) error {
		err := answer(pid, sig)
		mu.Lock()
		calls = append(calls, call{pid, sig, err})
		mu.Unlock()
		return err
	}
	t.Cleanup(func() { signalGroup = old })
	return func() []call {
		mu.Lock()
		defer mu.Unlock()
		return append([]call(nil), calls...)
	}
}

// ADR-095 T2. A stop runs once per command and sends nothing after the group
// was seen empty: a signal then can land on a reused id (#241). A cancelled
// Command whose group obeys TERM, then Run's reap: no call follows the first
// ESRCH, and the reap adds none.
func TestAReapAfterACancelSendsNothing(t *testing.T) {
	calls := recordSignals(t, syscall.Kill)
	pid := filepath.Join(t.TempDir(), "pid")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := Command(ctx, "sh", "-c", "echo $$ > \"$1\"; sleep 30 & wait", "sh", pid)
	done := make(chan error, 1)
	go func() { done <- Run(c) }()
	readPid(t, pid)
	cancel()
	<-done
	got := calls()
	if len(got) == 0 || got[0].sig != syscall.SIGTERM {
		t.Fatalf("the first signal is not TERM: %+v", got)
	}
	for i, x := range got {
		if errors.Is(x.err, syscall.ESRCH) {
			if i != len(got)-1 {
				t.Errorf("%d call(s) followed the first empty answer: %+v", len(got)-1-i, got)
			}
			return
		}
	}
	t.Errorf("the group was never seen empty: %+v", got)
}

// ADR-095 T2. Only ESRCH means the group is empty. A group answering EPERM to
// every poll is not empty: the stop waits out the grace from its TERM and then
// sends SIGKILL. A group answering ESRCH to the first poll gets nothing after.
func TestAnEPERMAnswerIsNotAnEmptyGroup(t *testing.T) {
	calls := recordSignals(t, func(pid int, sig syscall.Signal) error {
		if sig == 0 {
			return syscall.EPERM
		}
		return nil
	})
	start := time.Now()
	_ = stopGroup(424242)
	if d := time.Since(start); d < waitDelay {
		t.Errorf("an EPERM answer ended the stop after %s, before the %s grace", d, waitDelay)
	}
	got := calls()
	if len(got) == 0 || got[len(got)-1].sig != syscall.SIGKILL || got[0].sig != syscall.SIGTERM {
		t.Errorf("want TERM first and KILL last: %+v", got)
	}

	t.Run("an ESRCH answer ends it", func(t *testing.T) {
		calls := recordSignals(t, func(pid int, sig syscall.Signal) error {
			if sig == 0 {
				return syscall.ESRCH
			}
			return nil
		})
		_ = stopGroup(424242)
		if got := calls(); len(got) != 2 || got[0].sig != syscall.SIGTERM || got[1].sig != 0 {
			t.Errorf("want TERM and one poll, nothing after: %+v", got)
		}
	})
}
