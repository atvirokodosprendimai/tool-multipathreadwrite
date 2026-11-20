//go:build windows

package links

import (
	"errors"
	"syscall"
)

// windowsInvalidNames are the Win32 errors an Lstat gives for a name the
// system will not have: ERROR_INVALID_NAME, ERROR_BAD_PATHNAME,
// ERROR_FILENAME_EXCED_RANGE and ERROR_DIRECTORY. Not ERROR_NOT_READY (21) or
// any I/O error: those say nothing about the name (#711).
var windowsInvalidNames = []syscall.Errno{123, 161, 206, 267}

// anyErrno reports whether err is one of codes.
func anyErrno(err error, codes []syscall.Errno) bool {
	for _, c := range codes {
		if errors.Is(err, c) {
			return true
		}
	}
	return false
}

// Follow is true on Windows, where Go 1.23+ neither marks a junction as a
// symlink nor follows one in EvalSymlinks, so the walk has to (ADR-071).
const Follow = true

// platformInvalidName is a name Win32 refuses, as a "*" a shell left
// unexpanded (#711). ENOTDIR, ENAMETOOLONG and EILSEQ are Go-invented errnos
// Lstat never returns here, so they are not listed.
func platformInvalidName(err error) bool { return anyErrno(err, windowsInvalidNames) }

// windowsUnreachable is ERROR_BAD_NETPATH, which Go also reports as
// ErrNotExist: an unreachable network path, whose components may exist.
var windowsUnreachable = []syscall.Errno{53}

// platformUnreachable is a lookup that could not reach the place at all.
func platformUnreachable(err error) bool { return anyErrno(err, windowsUnreachable) }
