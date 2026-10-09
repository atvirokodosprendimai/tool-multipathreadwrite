package rooted

import (
	"os"
	"syscall"
)

// linkCount is the number of names of the file at path, read from an open
// handle: Windows does not put it in the FileInfo (ADR-134). A file that
// cannot be opened has an unknown count, which counts as "may be linked".
func linkCount(_ os.FileInfo, path string) (uint64, bool) {
	f, err := os.Open(path)
	if err != nil {
		return 0, false
	}
	defer f.Close() //nolint:errcheck // a read-only handle opened for one query
	var info syscall.ByHandleFileInformation
	if err := syscall.GetFileInformationByHandle(syscall.Handle(f.Fd()), &info); err != nil {
		return 0, false
	}
	return uint64(info.NumberOfLinks), true
}
