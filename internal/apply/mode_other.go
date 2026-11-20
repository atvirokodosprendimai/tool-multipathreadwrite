//go:build !windows

package apply

import (
	"os"
	"syscall"
)

// newFileMode is the mode of a file a plan creates: 0666 &^ umask, as open(2)
// gives one where no default ACL applies (#714; under a Linux default ACL open(2)
// ignores the umask and mrw does not, as before). The umask is read once, when
// the process starts, because the only way to read it is to set it, and that
// must not race a write.
var newFileMode = os.FileMode(0o666 &^ processUmask())

// processUmask is the process's umask, or 022 when the syscall answers with
// something that is not one (a sandbox that refuses umask returns -1): a new
// file is then 0644, as it was before #714, never 0000.
func processUmask() int {
	m := syscall.Umask(0)
	if m < 0 || m > 0o777 {
		return 0o022
	}
	syscall.Umask(m)
	return m
}
