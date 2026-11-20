//go:build !windows

package apply

import "syscall"

// The errors a filesystem that cannot hard-link answers, and one that is not a
// reason to fall back, as noHardLinks reads them on this OS.
var (
	errNoLinksA = syscall.EXDEV
	errNoLinksB = syscall.EPERM
)
