//go:build unix

package check

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/subproc"
)

// reapFixture is the check of #734's tests: a shell that exits 1 and leaves a
// child ignoring TERM, so its process group is reaped for a full second. The child
// writes a marker only after its trap is installed and the shell waits for it
// before exiting, so the group is never empty before the reap begins (the child
// racing its own trap let one run in 32 pass vacuously). gone waits until the
// shell itself has exited, and returns when it did.
func reapFixture(t *testing.T) (command string, gone func() time.Time) {
	t.Helper()
	dir := t.TempDir()
	marker, pidFile := filepath.Join(dir, "armed"), filepath.Join(dir, "pid")
	command = fmt.Sprintf(`echo $$ > %q; (trap "" TERM; : > %q; exec sleep 30) & while [ ! -e %q ]; do sleep 0.01; done; exit 1`, pidFile, marker, marker)
	gone = func() time.Time {
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			b, err := os.ReadFile(pidFile)
			if pid, perr := strconv.Atoi(strings.TrimSpace(string(b))); err == nil && perr == nil {
				if _, serr := os.Stat(marker); serr == nil && syscall.Kill(pid, 0) == syscall.ESRCH {
					return time.Now()
				}
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Error("the shell never exited")
		return time.Now()
	}
	return command, gone
}

// Issue #734. A check that exits 1 well inside its deadline, but leaves a child
// that ignores TERM, has its process group reaped for up to a second; a deadline
// that lands during that reap was relabelled "timed out" with exit -1 over the
// real exit status. The exit was observed before the deadline, so it stands.
func TestARealExitStatusSurvivesADeadlineThatLandsDuringTheReap(t *testing.T) {
	command, gone := reapFixture(t)
	start := time.Now()
	done := make(chan time.Time, 1)
	go func() { done <- gone() }()
	res, err := Run(context.Background(), t.TempDir(), Config{Check: command, TimeoutSeconds: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	exitedAt := <-done
	if d := exitedAt.Sub(start); d > 900*time.Millisecond {
		t.Fatalf("the shell exited %v in, too late for the 1s deadline to land in the reap: the fixture is not reaching it", d)
	}
	if res.DurationMS < 900 {
		t.Fatalf("the run took %dms: the reap was not what ended it", res.DurationMS)
	}
	if !res.Ran || res.ExitCode != 1 || res.Skipped != "" {
		t.Errorf("ran=%v exit=%d skipped=%q after %dms; want the check's own exit 1, not a timeout", res.Ran, res.ExitCode, res.Skipped, res.DurationMS)
	}
}

// A TERM or cancel that arrives while the process group is being reaped did not
// stop the command either: it had already exited 1 on its own. Its exit status
// stands, as ADR-177 reads "130": a run mrw stopped.
func TestARealExitStatusSurvivesAnInterruptThatLandsDuringTheReap(t *testing.T) {
	command, gone := reapFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var mu sync.Mutex
	var order []string
	record := func(e string) {
		mu.Lock()
		defer mu.Unlock()
		order = append(order, e)
	}
	subproc.Trace = record
	t.Cleanup(func() { subproc.Trace = nil })
	go func() {
		gone()
		time.Sleep(100 * time.Millisecond) // well inside the second the reap takes
		record("cancel")
		cancel()
	}()
	res, err := Run(ctx, t.TempDir(), Config{Check: command}, nil)
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	got := strings.Join(order, ",")
	mu.Unlock()
	if got != "exited,cancel,reaped" {
		t.Fatalf("order of events %q, want exited,cancel,reaped: the cancel did not land during the reap, so the test proves nothing", got)
	}
	if !res.Ran || res.ExitCode != 1 || res.Skipped != "" {
		t.Errorf("ran=%v exit=%d skipped=%q after %dms; want the command's own exit 1, not an interrupt", res.Ran, res.ExitCode, res.Skipped, res.DurationMS)
	}
}

// The pair: the same cancel while the command still runs stops it.
func TestACancelWhileTheCommandRunsIsStillAnInterrupt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		time.Sleep(400 * time.Millisecond)
		cancel()
	}()
	res, err := Run(ctx, t.TempDir(), Config{Check: `exec sleep 30`}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ran || res.ExitCode != -1 || res.Skipped != Interrupted {
		t.Errorf("ran=%v exit=%d skipped=%q; want interrupted", res.Ran, res.ExitCode, res.Skipped)
	}
}

// The pair for the deadline: a check still running at its deadline is timed out.
func TestACheckStillRunningAtItsDeadlineIsStillTimedOut(t *testing.T) {
	res, err := Run(context.Background(), t.TempDir(), Config{Check: `exec sleep 30`, TimeoutSeconds: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Ran || res.ExitCode != -1 || res.Skipped == "" {
		t.Errorf("ran=%v exit=%d skipped=%q; want timed out", res.Ran, res.ExitCode, res.Skipped)
	}
}
