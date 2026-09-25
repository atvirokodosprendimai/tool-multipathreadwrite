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
