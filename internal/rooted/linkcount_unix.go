//go:build unix

package rooted

import (
	"os"
	"syscall"
)

// linkCount is fi's number of names, from the Lstat already taken: no
// further syscall (ADR-134).
func linkCount(fi os.FileInfo, _ string) (uint64, bool) {
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		return uint64(st.Nlink), true //nolint:unconvert // Nlink is uint16 on darwin, uint64 on linux
	}
	return 0, false
}
