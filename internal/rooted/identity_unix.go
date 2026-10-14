//go:build unix

package rooted

import (
	"errors"
	"os"
	"syscall"
)

// identity is fi's file identity and number of names, from the Lstat already
// taken: no further syscall (ADR-134).
func identity(fi os.FileInfo, _ string) (fileKey, uint64, error) {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return fileKey{}, 0, errors.New("no file identity")
	}
	//nolint:unconvert // Dev and Nlink are narrower than uint64 on darwin
	return fileKey{uint64(st.Dev), uint64(st.Ino)}, uint64(st.Nlink), nil
}
