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

// ADR-080 (the waiver on #232). A child that exits 0 and leaves a grandchild
// behind — with its stdio redirected, so no pipe holds Wait — returned at once,
// and the grandchild outlived mrw: the group is killed only on a cancel. Run and
// Output reap the group after every exit.
func TestAGrandchildOfACleanExitIsReaped(t *testing.T) {
	for name, run := range map[string]func(ctx context.Context, script string) error{
		"Run": func(ctx context.Context, script string) error {
			return Run(Command(ctx, "sh", "-c", script))
		},
		"Output": func(ctx context.Context, script string) error {
			_, err := Output(Command(ctx, "sh", "-c", script))
			return err
		},
	} {
		pidFile := filepath.Join(t.TempDir(), "gc.pid")
		script := "sleep 30 >/dev/null 2>&1 & echo $! > '" + pidFile + "'; exit 0"
		if err := run(context.Background(), script); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !gone(t, pidFile) {
			t.Errorf("%s: the grandchild of a clean exit outlived it", name)
		}
	}
}

// gone waits up to three seconds for pid to stop existing, and kills it if it
// does not, so a failing test leaves nothing behind.
func gone(t *testing.T, pidFile string) bool {
	t.Helper()
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
			return false
		}
		time.Sleep(50 * time.Millisecond)
	}
	return true
}
