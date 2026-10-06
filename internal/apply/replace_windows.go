//go:build windows

package apply

import (
	"errors"
	"io/fs"
	"syscall"
)

// The Win32 errors a refused open of a target answers that no portable error
// names (ADR-125): another process holds the file without sharing what the
// open asked for, or the name is one the system will not make.
const (
	errorSharingViolation syscall.Errno = 32
	errorInvalidName      syscall.Errno = 123
)

// platformCause names a Windows sharing violation and an invalid name.
func platformCause(err error) string {
	switch {
	case errors.Is(err, errorSharingViolation):
		return "held open by another process"
	case errors.Is(err, errorInvalidName):
		return "not a valid name on this system"
	}
	return ""
}

// accessDelete is DELETE, the right a rename over a file or its removal
// needs; syscall names no constant for it.
const accessDelete = 0x00010000

// replaceable opens full for DELETE with every kind of sharing and closes it
// at once (ADR-125). A holder that did not share delete refuses the open with
// a sharing violation, exactly as it would refuse the commit's rename, and
// nothing has been written yet. The open neither follows a link at the end
// (it is the entry the rename replaces) nor changes anything on disk.
func replaceable(full string) error {
	p, err := syscall.UTF16PtrFromString(full)
	if err != nil {
		return &fs.PathError{Op: "open", Path: full, Err: err}
	}
	h, err := syscall.CreateFile(p, accessDelete,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
		nil, syscall.OPEN_EXISTING,
		syscall.FILE_FLAG_BACKUP_SEMANTICS|syscall.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return &fs.PathError{Op: "open", Path: full, Err: err}
	}
	_ = syscall.CloseHandle(h)
	return nil
}
