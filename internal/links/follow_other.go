//go:build !windows

package links

import (
	"errors"
	"syscall"
)

// Follow is false where EvalSymlinks already follows every link, so Real and
// the boundary need no walk of their own (ADR-071).
const Follow = false

// platformInvalidName is a name the system refuses: the parent is a file, the
// name is too long, or APFS refuses its bytes (ADR-132). The walk runs here
// only under test; on Windows these errnos are Go-invented and never returned.
func platformInvalidName(err error) bool {
	return errors.Is(err, syscall.ENOTDIR) || errors.Is(err, syscall.ENAMETOOLONG) || errors.Is(err, syscall.EILSEQ)
}
