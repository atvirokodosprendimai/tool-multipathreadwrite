//go:build unix

package subproc

import (
	"errors"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// signalGroup is how every signal subproc sends a process group leaves it:
// syscall.Kill, given the negated group id. A test replaces it to record or
// answer the calls (ADR-095).
var signalGroup = syscall.Kill

// pollEvery is how often stopGroup asks whether the group has emptied.
const pollEvery = 5 * time.Millisecond

// group starts c as the leader of a new process group, and makes a cancel stop
// that whole group with stopGroup, once: a grandchild stops with the child mrw
// started, and a nested mrw hears TERM in time to stop its own check.
func group(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var once sync.Once
	var err error
	c.Cancel = func() error {
		once.Do(func() { err = stopGroup(c.Process.Pid) })
		return err
	}
}

// reap stops c's process group once c has exited, taking any grandchild still
// in it (ADR-080). It is the cancel's own once-only stop: after a cancel it
// sends nothing, since the group was already seen empty or killed, and after a
// clean exit its TERM finds an empty group and nothing else is sent.
func reap(c *exec.Cmd) {
	if c.Process != nil && c.Cancel != nil {
		_ = c.Cancel()
	}
}

// run runs c, then reaps its process group (ADR-080).
func run(c *exec.Cmd) error {
	err := c.Run()
	reap(c)
	return err
}

// stopGroup stops process group pgid (ADR-095): SIGTERM, then a poll every
// pollEvery until no member is left or waitDelay has passed since the TERM,
// then SIGKILL to what is left. Only ESRCH means empty; any other answer,
// EPERM included, is a member left. Nothing is sent once the group was seen
// empty, since its id could then be reused (the review of #241). It returns
// the TERM's error.
func stopGroup(pgid int) error {
	termErr := signalGroup(-pgid, syscall.SIGTERM)
	if errors.Is(termErr, syscall.ESRCH) {
		return termErr
	}
	deadline := time.Now().Add(waitDelay)
	for time.Now().Before(deadline) {
		if errors.Is(signalGroup(-pgid, 0), syscall.ESRCH) {
			return termErr
		}
		time.Sleep(pollEvery)
	}
	_ = signalGroup(-pgid, syscall.SIGKILL)
	return termErr
}
