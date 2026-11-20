//go:build windows

package links

import "syscall"

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
