//go:build windows

package rooted

import (
	"strings"
	"syscall"
)

// followLinks is true on Windows, where Go 1.23+ neither marks a junction as a
// symlink nor follows one in EvalSymlinks, and where Win32 reads a name that
// ends in a dot or a space as another (ADR-071).
const followLinks = true

// opensDevice reports whether Windows opens full as a device rather than a
// file. GetFullPathName, which CreateFile applies first, answers \\.\NAME for
// a device; asking it rather than a list keeps nul.txt a file on Windows 11
// and a device before it (ADR-076).
func opensDevice(full string) bool {
	p, err := syscall.FullPath(full)
	return err == nil && strings.HasPrefix(p, `\\.\`)
}
