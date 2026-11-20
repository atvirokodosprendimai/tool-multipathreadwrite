//go:build !windows

package rooted

import (
	"errors"
	"io/fs"
)

// fallsBack reports a refusal of os.Root that opening by name may still answer:
// a directory it may search but not read (#727).
func fallsBack(err error) bool { return errors.Is(err, fs.ErrPermission) }
