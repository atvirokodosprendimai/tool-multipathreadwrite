//go:build !windows

package rooted

import (
	"errors"
	"os"
	"syscall"
)

// dirBudget is how many directory handles a walk may keep open: what is left of
// the process's descriptor limit once the scans in flight, the root and the
// standard streams are provided for, a quarter of it at most, and 32 whatever
// the limit (#727).
func dirBudget() int {
	var l syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &l); err != nil {
		return 8
	}
	soft := int(l.Cur)
	if l.Cur > 1<<20 {
		soft = 1 << 20
	}
	return min(32, max(0, (soft-40)/4))
}

// fdExhausted reports a refusal for want of descriptors.
func fdExhausted(err error) bool {
	return errors.Is(err, syscall.EMFILE) || errors.Is(err, syscall.ENFILE)
}

// openDirProbe opens dir through r as a directory, without blocking, and closes
// it: true when it is one. A FIFO fails O_DIRECTORY at once, where OpenRoot would
// wait for a writer (#727).
func openDirProbe(r *os.Root, dir string) bool {
	f, err := r.OpenFile(dir, os.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}
