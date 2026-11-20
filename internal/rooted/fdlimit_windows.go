//go:build windows

package rooted

import (
	"errors"
	"os"
	"syscall"
)

// dirBudget is how many directory handles a walk may keep open (#727); Windows
// has no descriptor limit of the kind unix does.
func dirBudget() int { return 32 }

// fdExhausted reports a refusal for want of descriptors.
func fdExhausted(err error) bool {
	return errors.Is(err, syscall.EMFILE) || errors.Is(err, syscall.ENFILE)
}

// openDirProbe says whether dir, through r, is a directory (#727).
func openDirProbe(r *os.Root, dir string) bool {
	fi, err := r.Stat(dir)
	return err == nil && fi.IsDir()
}
