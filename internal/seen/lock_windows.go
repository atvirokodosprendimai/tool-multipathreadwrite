//go:build windows

package seen

import (
	"os"
	"syscall"
	"unsafe"
)

// LockFileEx lives in kernel32, not in syscall. LazyDLL keeps go.mod at one
// require (ADR-038).
const lockfileExclusiveLock = 2

var (
	modkernel32      = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = modkernel32.NewProc("LockFileEx")
	procUnlockFileEx = modkernel32.NewProc("UnlockFileEx")
)

func lock(f *os.File) error {
	var ov syscall.Overlapped
	r1, _, err := procLockFileEx.Call(f.Fd(), uintptr(lockfileExclusiveLock), 0, 1, 0, uintptr(unsafe.Pointer(&ov)))
	if r1 == 0 {
		return err
	}
	return nil
}

func unlock(f *os.File) error {
	var ov syscall.Overlapped
	r1, _, err := procUnlockFileEx.Call(f.Fd(), 0, 1, 0, uintptr(unsafe.Pointer(&ov)))
	if r1 == 0 {
		return err
	}
	return nil
}
