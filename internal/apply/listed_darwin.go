//go:build darwin

package apply

import (
	"bytes"
	"path/filepath"
	"syscall"
	"unsafe"
)

// spelledByDescriptor is the name the descriptor's path gives target (F_GETPATH), and
// whether target has exactly one name. With one name that is the name the volume
// stores. With several the path may be any of them: a name equal to the plan's
// spelling still proves the plan's spelling is listed, and any other is not to be
// trusted, so the caller lists the directory. It costs one open and one fcntl where
// listing a directory of a hundred thousand entries cost a third of a second
// (ADR-209, measured).
func spelledByDescriptor(tr *tree, target string) (string, bool, bool) {
	f, fi, err := tr.openRead(target)
	if err != nil {
		return "", false, false
	}
	defer func() { _ = f.Close() }()
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return "", false, false
	}
	var buf [1024]byte // MAXPATHLEN
	if _, _, e := syscall.Syscall(syscall.SYS_FCNTL, f.Fd(), syscall.F_GETPATH, uintptr(unsafe.Pointer(&buf[0]))); e != 0 {
		return "", false, false
	}
	n := bytes.IndexByte(buf[:], 0)
	if n <= 0 {
		return "", false, false
	}
	return filepath.Base(string(buf[:n])), st.Nlink == 1, true
}
