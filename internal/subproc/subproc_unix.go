//go:build unix

package subproc

import (
	"os/exec"
	"syscall"
)

// group starts c as the leader of a new process group and makes a cancel kill
// that whole group, so a grandchild dies with the child mrw started.
func group(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	c.Cancel = func() error {
		return syscall.Kill(-c.Process.Pid, syscall.SIGKILL)
	}
}

// reap kills c's process group once c has exited, taking any grandchild still
// in it. A group id is kept while a member lives; once every member is gone
// the kill finds none, and the id could be reused in the microseconds before
// it lands — the one window this leaves.
func reap(c *exec.Cmd) {
	if c.Process != nil {
		_ = syscall.Kill(-c.Process.Pid, syscall.SIGKILL)
	}
}
