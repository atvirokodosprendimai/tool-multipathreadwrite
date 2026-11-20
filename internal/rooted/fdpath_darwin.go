//go:build darwin

package rooted

import (
	"os"
	"syscall"
	"unsafe"
)

// fdPath is the path the open descriptor f names now, as the system reports it
// (F_GETPATH, #727): the opened file itself says where it is, rather than a
// second lookup of a name.
func fdPath(f *os.File) (string, bool) {
	const fGetPath = 50
	var buf [1024]byte
	if _, _, e := syscall.Syscall(syscall.SYS_FCNTL, f.Fd(), fGetPath, uintptr(unsafe.Pointer(&buf[0]))); e != 0 {
		return "", false
	}
	n := 0
	for n < len(buf) && buf[n] != 0 {
		n++
	}
	return string(buf[:n]), n > 0
}
