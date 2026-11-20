//go:build windows

package apply

import (
	"errors"
	"syscall"
)

// The Win32 errors a hard link answers when the filesystem has none or the
// target is on another volume: ERROR_INVALID_FUNCTION (FAT, exFAT),
// ERROR_NOT_SAME_DEVICE and ERROR_NOT_SUPPORTED. Access denied is not here: an
// ACL that refuses the link is the move's, and the name is not known to be free.
const (
	errorInvalidFunction syscall.Errno = 1
	errorNotSameDevice   syscall.Errno = 17
	errorNotSupported    syscall.Errno = 50
)

// noHardLinks says the OS refused a hard link because the filesystem cannot
// have one, not for a reason that is the move's own.
func noHardLinks(err error) bool {
	for _, e := range []error{errorInvalidFunction, errorNotSameDevice, errorNotSupported} {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}
