//go:build windows

package state

import (
	"errors"
	"os"
	"syscall"
	"unsafe"
)

// LockFileEx lives in kernel32, not in syscall. LazyDLL keeps go.mod at one
// require (ADR-038).
const lockfileExclusiveLock = 2

// lockfileFailImmediately makes LockFileEx return at once when the range is
// held, with ERROR_LOCK_VIOLATION, instead of waiting (ADR-110).
const lockfileFailImmediately = 1

// errorLockViolation is the Win32 ERROR_LOCK_VIOLATION.
const errorLockViolation = syscall.Errno(33)

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

// tryLock takes f's lock if it is free and reports whether it did; another
// holder is (false, nil), never an error (ADR-110).
func tryLock(f *os.File) (bool, error) {
	var ov syscall.Overlapped
	r1, _, err := procLockFileEx.Call(f.Fd(), uintptr(lockfileExclusiveLock|lockfileFailImmediately), 0, 1, 0, uintptr(unsafe.Pointer(&ov)))
	if r1 != 0 {
		return true, nil
	}
	if errors.Is(err, errorLockViolation) {
		return false, nil
	}
	return false, err
}
