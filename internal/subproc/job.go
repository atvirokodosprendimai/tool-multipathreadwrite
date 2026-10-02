//go:build windows || mrwjobs

package subproc

import (
	"fmt"
	"os/exec"
	"sync"
)

// jobObjectLimitKillOnJobClose is JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE: closing a
// job's last handle kills every process in it, including when mrw itself dies.
const jobObjectLimitKillOnJobClose = 0x2000

// basicLimitInformation, ioCounters and extendedLimitInformation are
// JOBOBJECT_BASIC_LIMIT_INFORMATION, IO_COUNTERS and
// JOBOBJECT_EXTENDED_LIMIT_INFORMATION, declared by hand because go.mod takes
// no golang.org/x/sys. TestTheJobLimitStructHasTheWin32Size pins the layout.
type basicLimitInformation struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
}

type ioCounters struct {
	ReadOperationCount, WriteOperationCount, OtherOperationCount uint64
	ReadTransferCount, WriteTransferCount, OtherTransferCount    uint64
}

type extendedLimitInformation struct {
	BasicLimitInformation basicLimitInformation
	IoInfo                ioCounters
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

// killOnCloseLimits is the only limit a job carries: kill every member when
// the job is closed. JOB_OBJECT_LIMIT_BREAKAWAY_OK is deliberately absent:
// Git for Windows' sh asks to break away whenever a job allows it, and its
// children then survive the job (a Windows peer, 2026-10-02; Zy: "Drop
// BREAKAWAY_OK").
func killOnCloseLimits() extendedLimitInformation {
	var l extendedLimitInformation
	l.BasicLimitInformation.LimitFlags = jobObjectLimitKillOnJobClose
	return l
}

// jobAPI is what runInJob needs from Windows' job objects. It is an interface
// so the sequence is driven by a fake on any platform; subproc_windows.go
// implements it with kernel32.
type jobAPI interface {
	create() (uintptr, error)
	assign(job uintptr, pid int) error
	resume(pid int) error
	terminate(job uintptr) error
	close(job uintptr) error
}

// jobTree is one child's job, shared by its Cancel and runInJob. job is zero
// until the child is assigned to it.
type jobTree struct {
	api jobAPI
	mu  sync.Mutex
	job uintptr
}

// cancel stops the child's whole tree: before the child is in its job, kill
// stops the child itself (still suspended, it has started nothing); after,
// the job is terminated, taking every member.
func (t *jobTree) cancel(kill func() error) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.job != 0 {
		return t.api.terminate(t.job)
	}
	return kill()
}

// runInJob starts c, which group made start suspended, assigns it to a fresh
// job, and only then resumes it, so a grandchild it starts at once is created
// inside the job. When c exits the job is terminated and closed, taking a
// grandchild left behind (ADR-080's reap). If the child cannot be contained it
// does not run uncontained: it is killed, the job closed, and the error
// returned (ADR-120).
func runInJob(c *exec.Cmd, t *jobTree) error {
	if err := c.Start(); err != nil {
		return err
	}
	job, err := t.api.create()
	if err == nil {
		t.mu.Lock()
		if err = t.api.assign(job, c.Process.Pid); err == nil {
			t.job = job
		}
		t.mu.Unlock()
		if err == nil {
			err = t.api.resume(c.Process.Pid)
		}
	}
	if err != nil {
		_ = c.Process.Kill()
		_ = c.Wait()
		t.mu.Lock()
		if job != 0 {
			_ = t.api.close(job)
		}
		t.job = 0
		t.mu.Unlock()
		// The child was killed while still suspended: it ran no instruction,
		// so it has no exit status to report. Clearing what Wait recorded is
		// what lets the check say it could not run, instead of reading the
		// kill's exit code as a failed check (the review of #325).
		c.ProcessState = nil
		return fmt.Errorf("could not contain the child in a job object: %w", err)
	}
	waitErr := c.Wait()
	t.mu.Lock()
	_ = t.api.terminate(job)
	_ = t.api.close(job)
	t.job = 0
	t.mu.Unlock()
	return waitErr
}
