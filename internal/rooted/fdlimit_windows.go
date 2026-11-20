//go:build windows

package rooted

import (
	"errors"
	"syscall"
)

// dirBudget is how many directory handles a walk may keep open (#727); Windows
// has no descriptor limit of the kind unix does.
func dirBudget() int { return 32 }

// fdExhausted reports a refusal for want of descriptors.
func fdExhausted(err error) bool {
	return errors.Is(err, syscall.EMFILE) || errors.Is(err, syscall.ENFILE)
}
