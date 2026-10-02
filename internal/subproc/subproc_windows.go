//go:build windows

package subproc

import (
	"fmt"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"
)

// kernel32 is reached through LazyDLL, as internal/state and internal/apply
// already do: go.mod keeps its one requirement (ADR-120).
var (
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	procCreateJobObjectW         = kernel32.NewProc("CreateJobObjectW")
	procSetInformationJobObject  = kernel32.NewProc("SetInformationJobObject")
	procAssignProcessToJobObject = kernel32.NewProc("AssignProcessToJobObject")
	procTerminateJobObject       = kernel32.NewProc("TerminateJobObject")
	procCreateToolhelp32Snapshot = kernel32.NewProc("CreateToolhelp32Snapshot")
	procThread32First            = kernel32.NewProc("Thread32First")
	procThread32Next             = kernel32.NewProc("Thread32Next")
	procOpenThread               = kernel32.NewProc("OpenThread")
	procResumeThread             = kernel32.NewProc("ResumeThread")
)

const (
	createSuspended                      = 0x00000004
	jobObjectExtendedLimitInformationKey = 9
	processSetQuota                      = 0x0100
	processTerminate                     = 0x0001
	th32csSnapThread                     = 0x00000004
	threadSuspendResume                  = 0x0002
)

// threadEntry32 is THREADENTRY32.
type threadEntry32 struct {
	Size           uint32
	Usage          uint32
	ThreadID       uint32
	OwnerProcessID uint32
	BasePri        int32
	DeltaPri       int32
	Flags          uint32
}

// trees holds each Command's job until Run takes it.
var trees sync.Map // *exec.Cmd → *jobTree

// group makes c start suspended and a cancel stop its job (ADR-120).
func group(c *exec.Cmd) {
	t := &jobTree{api: kernel32Jobs{}}
	if c.SysProcAttr == nil {
		c.SysProcAttr = &syscall.SysProcAttr{}
	}
	c.SysProcAttr.CreationFlags |= createSuspended
	c.Cancel = func() error { return t.cancel(c.Process.Kill) }
	trees.Store(c, t)
}

// run runs a Command's child inside its job; any other Cmd runs as it is.
func run(c *exec.Cmd) error {
	v, ok := trees.LoadAndDelete(c)
	if !ok {
		return c.Run()
	}
	return runInJob(c, v.(*jobTree))
}

// kernel32Jobs is jobAPI on Windows.
type kernel32Jobs struct{}

func (kernel32Jobs) create() (uintptr, error) {
	h, _, err := procCreateJobObjectW.Call(0, 0)
	if h == 0 {
		return 0, fmt.Errorf("CreateJobObject: %w", err)
	}
	l := killOnCloseLimits()
	if r, _, err := procSetInformationJobObject.Call(h, jobObjectExtendedLimitInformationKey,
		uintptr(unsafe.Pointer(&l)), unsafe.Sizeof(l)); r == 0 {
		_ = syscall.CloseHandle(syscall.Handle(h))
		return 0, fmt.Errorf("SetInformationJobObject: %w", err)
	}
	return h, nil
}

func (kernel32Jobs) assign(job uintptr, pid int) error {
	p, err := syscall.OpenProcess(processSetQuota|processTerminate, false, uint32(pid))
	if err != nil {
		return fmt.Errorf("OpenProcess: %w", err)
	}
	defer func() { _ = syscall.CloseHandle(p) }()
	if r, _, err := procAssignProcessToJobObject.Call(job, uintptr(p)); r == 0 {
		return fmt.Errorf("AssignProcessToJobObject: %w", err)
	}
	return nil
}

// resume resumes every thread of a process started suspended; it has one,
// found through a Toolhelp32 snapshot since exec.Cmd keeps no thread handle.
func (kernel32Jobs) resume(pid int) error {
	snap, _, err := procCreateToolhelp32Snapshot.Call(th32csSnapThread, 0)
	if syscall.Handle(snap) == syscall.InvalidHandle {
		return fmt.Errorf("CreateToolhelp32Snapshot: %w", err)
	}
	defer func() { _ = syscall.CloseHandle(syscall.Handle(snap)) }()
	e := threadEntry32{Size: uint32(unsafe.Sizeof(threadEntry32{}))}
	resumed := 0
	for r, _, _ := procThread32First.Call(snap, uintptr(unsafe.Pointer(&e))); r != 0; r, _, _ = procThread32Next.Call(snap, uintptr(unsafe.Pointer(&e))) {
		if e.OwnerProcessID != uint32(pid) {
			continue
		}
		th, _, err := procOpenThread.Call(threadSuspendResume, 0, uintptr(e.ThreadID))
		if th == 0 {
			return fmt.Errorf("OpenThread: %w", err)
		}
		n, _, err := procResumeThread.Call(th)
		_ = syscall.CloseHandle(syscall.Handle(th))
		if int32(n) == -1 {
			return fmt.Errorf("ResumeThread: %w", err)
		}
		resumed++
	}
	if resumed == 0 {
		return fmt.Errorf("no thread of process %d to resume", pid)
	}
	return nil
}

func (kernel32Jobs) terminate(job uintptr) error {
	if r, _, err := procTerminateJobObject.Call(job, 1); r == 0 {
		return fmt.Errorf("TerminateJobObject: %w", err)
	}
	return nil
}

func (kernel32Jobs) close(job uintptr) error {
	return syscall.CloseHandle(syscall.Handle(job))
}
