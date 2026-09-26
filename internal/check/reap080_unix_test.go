//go:build unix

package check

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

// ADR-080 (M: reap always). A check that passes and leaves a background process
// running — its stdio redirected, so nothing waits on it — left that process
// behind after mrw returned. The check's group is reaped after every exit.
func TestACheckThatPassesLeavesNoProcessBehind(t *testing.T) {
	root := t.TempDir()
	res, err := Run(context.Background(), root,
		Config{Check: `sleep 30 >/dev/null 2>&1 & echo $! > gc.pid; exit 0`, declared: true}, nil)
	if err != nil || !res.OK() {
		t.Fatalf("the check did not pass: %v %+v", err, res)
	}
	if !gone(t, filepath.Join(root, "gc.pid")) {
		t.Error("a passing check's background process outlived it")
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
