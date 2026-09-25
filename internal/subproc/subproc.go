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
package subproc

import (
	"context"
	"os/exec"
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
