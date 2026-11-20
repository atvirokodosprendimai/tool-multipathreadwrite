//go:build windows

package links

import (
	"errors"
	"syscall"
)

// Follow is true on Windows, where Go 1.23+ neither marks a junction as a
// symlink nor follows one in EvalSymlinks, so the walk has to (ADR-071).
const Follow = true

// platformInvalidName names the Windows answers for a name the system will not
// have, as a "*" a shell left unexpanded: ERROR_INVALID_NAME,
// ERROR_BAD_PATHNAME, ERROR_FILENAME_EXCED_RANGE and ERROR_DIRECTORY (#711).
func platformInvalidName(err error) bool {
	return errors.Is(err, syscall.Errno(123)) || errors.Is(err, syscall.Errno(161)) ||
		errors.Is(err, syscall.Errno(206)) || errors.Is(err, syscall.Errno(267))
}
