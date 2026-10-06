//go:build !windows

package apply

import (
	"errors"
	"syscall"
)

// platformCause names a name the system refuses (ADR-132): APFS refuses a byte
// that is not UTF-8 with EILSEQ, and every unix a name too long with
// ENAMETOOLONG. A permission is causeOf's; nothing else is named here.
func platformCause(err error) string {
	if errors.Is(err, syscall.EILSEQ) || errors.Is(err, syscall.ENAMETOOLONG) {
		return "not a valid name on this system"
	}
	return ""
}

// replaceable answers yes on unix: a rename over a file, or its removal, does
// not care who has it open.
func replaceable(*tree, string) error { return nil }
