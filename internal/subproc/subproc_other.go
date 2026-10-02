//go:build !unix && !windows

package subproc

import (
	"context"
	"os/exec"
)

// group does nothing where there are no process groups to kill: only the
// bound on the wait for held pipes applies (ADR-072).
func group(_ context.Context, c *exec.Cmd) {}

// run runs c; there is no group to reap afterwards (ADR-080).
func run(c *exec.Cmd) error {
	return c.Run()
}
