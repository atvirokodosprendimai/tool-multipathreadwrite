//go:build !windows

package apply

import (
	"errors"
	"syscall"
)

// noHardLinks says the OS refused a hard link because the entry or the
// filesystem cannot have one, not for a reason that is the move's own: a
// directory or protected_hardlinks (EPERM), a filesystem without hard links
// (EPERM, ENOTSUP, EOPNOTSUPP, ENOSYS), another device (EXDEV), too many links
// (EMLINK).
func noHardLinks(err error) bool {
	for _, e := range []error{syscall.EPERM, syscall.ENOTSUP, syscall.EOPNOTSUPP, syscall.ENOSYS, syscall.EXDEV, syscall.EMLINK} {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}
