// Package subproc starts a child process that a deadline or a cancel actually
// stops.
//
// exec.CommandContext kills only the process it started, and exec.Cmd waits for
// every pipe to close. So a check run as `sh -c 'make test'` left make running
// after mrw said "timed out", and an ast-grep behind a wrapper script held its
// stdout past the 2 s bound for as long as its grandchild lived (ADR-072, from
// the v1.25.1 adversarial round). Command puts the child in a process group of
// its own on unix and kills the whole group on cancel, and on every platform it
// bounds how long Wait waits for pipes a grandchild still holds. On Windows only
// that bound applies: a grandchild there can outlive the kill.
// Interruptible listens, for such a child, for the ^C, terminate or hangup its
// own process group no longer hears, and cancels the context it runs under, so
// its group is killed (ADR-074).
package subproc

import (
	"context"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

// waitDelay is how long Wait waits, after the child is killed, for pipes a
// grandchild still holds before it closes them itself.
const waitDelay = time.Second

// Command is exec.CommandContext for a child whose descendants must stop with
// it. The child's stdin is left nil (the null device): a child in its own
// process group must not read the terminal.
func Command(ctx context.Context, name string, args ...string) *exec.Cmd {
	c := exec.CommandContext(ctx, name, args...)
	c.WaitDelay = waitDelay
	group(c)
	return c
}

// Signals are the signals that stop a child Command started while the caller
// waits on it: an interrupt, a terminate and a hangup, each of which would
// otherwise end mrw and leave the child, in a process group of its own,
// running. A signal the process was started with IGNORED is left out and stays
// ignored: nohup ignores SIGHUP and a shell starts a background job with SIGINT
// ignored, and signal.Notify would switch either back on (ADR-072, review of
// #229; moved here from the check by ADR-074).
func Signals() []os.Signal {
	var out []os.Signal
	for _, s := range []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGHUP} {
		if !signal.Ignored(s) {
			out = append(out, s)
		}
	}
	return out
}

// Interruptible returns a context that any of Signals cancels, and the
// function that stops listening and gives each signal back its default. Run a
// Command under it for exactly as long as the child runs, so a ^C anywhere else
// behaves as it always did.
func Interruptible(ctx context.Context) (context.Context, context.CancelFunc) {
	sigs := Signals()
	if len(sigs) == 0 {
		// NotifyContext naming no signal relays EVERY signal the process gets.
		return ctx, func() {}
	}
	return signal.NotifyContext(ctx, sigs...)
}

// Run runs c and then kills whatever is left of its process group: a child
// that exited 0 could leave a background grandchild behind, and exec.Cmd
// cancels the group only on a deadline or a cancel, so the grandchild outlived
// mrw (the waiver on #232). mrw started the group, and nothing it started
// outlives the call (ADR-080, M: reap always). A grandchild that called setsid
// is in a group of its own and escapes, as it would a shell.
func Run(c *exec.Cmd) error {
	err := c.Run()
	reap(c)
	return err
}

// Output is Run for a child whose stdout is the answer. The answer goes to a
// file, not a pipe: exec waits up to WaitDelay for a pipe a grandchild still
// holds, and the group was killed only after that — up to a second past the
// child's exit, in which an emptied group's id could be reused (the review of
// #241). With a file, Wait returns at the child's exit and the group is killed
// at once.
func Output(c *exec.Cmd) ([]byte, error) {
	f, err := os.CreateTemp("", "mrw-subproc-*.out")
	if err != nil {
		return nil, err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	c.Stdout = f
	runErr := Run(c)
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	out, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return out, runErr
}
