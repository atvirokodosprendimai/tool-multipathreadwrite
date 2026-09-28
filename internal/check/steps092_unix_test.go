//go:build unix

package check

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
	"time"
)

// ADR-092. One interrupt handler covers the whole sequence. DURING: an
// interrupt while step 2 runs stops it, and step 3 never runs. BETWEEN: an
// interrupt after step 1 and before step 2 starts is still caught — with a
// handler scoped to one child it would be unhandled, and the default action
// would kill this test.
func TestAnInterruptDuringASequenceStopsItAndNamesTheRest(t *testing.T) {
	if signal.Ignored(syscall.SIGINT) {
		t.Skip("this test process was started with SIGINT ignored")
	}
	t.Run("during", func(t *testing.T) {
		tempDirForLogs(t)
		root := t.TempDir()
		done := make(chan struct{})
		go func() {
			for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
				select {
				case <-done:
					return
				default:
				}
				if _, err := os.Stat(filepath.Join(root, "started2")); err == nil {
					_ = syscall.Kill(os.Getpid(), syscall.SIGINT)
					return
				}
			}
		}()
		res := RunSteps(context.Background(), root, Config{}, []Step{
			{Name: "one", Command: "exit 0"},
			{Name: "two", Command: "touch started2; exec sleep 30"},
			{Name: "three", Command: "touch m3"},
		})
		close(done)
		if got, want := statuses(res), []string{StepPass, StepInterrupted, StepNotRun}; !reflect.DeepEqual(got, want) {
			t.Fatalf("statuses %v, want %v", got, want)
		}
		if !res.Steps[1].Ran {
			t.Errorf("a step stopped while it ran says it did not run: %+v", res.Steps[1])
		}
		if _, err := os.Stat(filepath.Join(root, "m3")); err == nil {
			t.Error("step 3 ran after the interrupt")
		}
	})
	t.Run("between", func(t *testing.T) {
		tempDirForLogs(t)
		root := t.TempDir()
		prev := afterStep
		t.Cleanup(func() { afterStep = prev })
		afterStep = func(ctx context.Context, i int) {
			if i != 0 {
				return
			}
			_ = syscall.Kill(os.Getpid(), syscall.SIGINT)
			select {
			case <-ctx.Done():
			case <-time.After(3 * time.Second):
				t.Error("an interrupt between steps did not end the sequence")
			}
		}
		res := RunSteps(context.Background(), root, Config{}, []Step{
			{Name: "one", Command: "exit 0"},
			{Name: "two", Command: "touch m2"},
			{Name: "three", Command: "touch m3"},
		})
		if got, want := statuses(res), []string{StepPass, StepInterrupted, StepNotRun}; !reflect.DeepEqual(got, want) {
			t.Fatalf("statuses %v, want %v", got, want)
		}
		if res.Steps[1].Ran {
			t.Errorf("a step interrupted before it started says it ran: %+v", res.Steps[1])
		}
		for _, m := range []string{"m2", "m3"} {
			if _, err := os.Stat(filepath.Join(root, m)); err == nil {
				t.Errorf("%s was written after the interrupt", m)
			}
		}
	})
}
