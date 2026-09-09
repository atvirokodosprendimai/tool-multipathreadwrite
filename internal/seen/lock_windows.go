//go:build windows

package seen

import (
	"os"
	"syscall"
)

const lockfileExclusiveLock = 2

func lock(f *os.File) error {
	var ov syscall.Overlapped
	return syscall.LockFileEx(syscall.Handle(f.Fd()), lockfileExclusiveLock, 0, 1, 0, &ov)
}

func unlock(f *os.File) error {
	var ov syscall.Overlapped
	return syscall.UnlockFileEx(syscall.Handle(f.Fd()), 0, 1, 0, &ov)
}
