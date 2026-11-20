//go:build windows

package rooted

import (
	"errors"
	"io/fs"
	"syscall"
)

// fallsBack reports a refusal of os.Root that opening by name may still answer:
// a permission refusal, or the reparse-point error OBJ_DONT_REPARSE gives on a
// reparse point that is no link (OneDrive, deduplication) (#727). Untested on
// Windows.
func fallsBack(err error) bool {
	return errors.Is(err, fs.ErrPermission) || errors.Is(err, syscall.ELOOP) || errors.Is(err, syscall.Errno(1920))
}
