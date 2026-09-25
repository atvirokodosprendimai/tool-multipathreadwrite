//go:build unix

package subproc

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

// ADR-074 T1. A child in a process group of its own never hears the terminal's
// ^C or a hangup, so the caller listens for it: a signal sent to mrw cancels the
// context the child runs under, and the cancel kills the child's group.
func TestASignalToMrwCancelsTheChildsContext(t *testing.T) {
	for _, s := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP} {
		if signal.Ignored(s) {
			continue // this test process was started with it ignored; it stays so
		}
		ctx, stop := Interruptible(context.Background())
		if err := syscall.Kill(os.Getpid(), s); err != nil {
			stop()
			t.Fatal(err)
		}
		select {
		case <-ctx.Done():
		case <-time.After(2 * time.Second):
			stop()
			t.Fatalf("%v sent to mrw did not cancel the child's context", s)
		}
		stop()
	}
}
