//go:build !unix

package subproc

import "os/exec"

// group does nothing where there are no process groups to kill: only the
// bound on the wait for held pipes applies (ADR-072).
func group(c *exec.Cmd) {}

// reap has no group to kill where there are no process groups: a grandchild
// there can outlive the child (ADR-080; a Windows job object is deferred).
func reap(c *exec.Cmd) {}
