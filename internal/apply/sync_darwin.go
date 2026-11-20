//go:build darwin

package apply

import (
	"errors"
	"os"
	"syscall"
)

// On macOS File.Sync is fcntl(F_FULLFSYNC), a drain of the device's queue, about
// 4 ms each (ADR-210). A plain fsync per file and one F_FULLFSYNC per device
// after them give every file that guarantee for one drain, since F_FULLFSYNC is
// a barrier for what was fsynced before it on the same device.
const (
	drains    = true
	syncsDirs = true
	// dirOpenFlags open a directory for a sync: only a directory, never blocking.
	dirOpenFlags = syscall.O_DIRECTORY | syscall.O_NONBLOCK
)

func platformSyncTemp(f *os.File) error { return fsyncRetrying(f) }

// fsyncRetrying is fsync, tried again when a signal interrupts it, as Go's own
// File.Sync does: an EINTR is not a failure to make the data stable.
func fsyncRetrying(f *os.File) error {
	return retryEINTR(func() error { return syscall.Fsync(int(f.Fd())) })
}

// retryEINTR runs fn until it returns anything but EINTR.
func retryEINTR(fn func() error) error {
	for {
		if err := fn(); !errors.Is(err, syscall.EINTR) {
			return err
		}
	}
}

// platformDrain issues F_FULLFSYNC itself. Go's File.Sync does the same but falls
// back to a plain fsync when the filesystem answers ENOTSUP (golang/go#64215), so
// a drain that did not drain would pass as one. Here ENOTSUP is an error, as ADR-210
// Decision 5 treats a mount that refuses a sync: before the first rename the write
// stops with nothing written, after the renames it is a landed write with a sync
// error. EINTR is retried.
func platformDrain(f *os.File) error {
	return retryEINTR(func() error { return fullFsync(f.Fd()) })
}

// fullFsync is fcntl(fd, F_FULLFSYNC), a variable so a test can answer ENOTSUP.
var fullFsync = func(fd uintptr) error {
	if _, _, e := syscall.Syscall(syscall.SYS_FCNTL, fd, syscall.F_FULLFSYNC, 0); e != 0 {
		return e
	}
	return nil
}

func platformSyncDir(f *os.File) error { return fsyncRetrying(f) }

func platformDevice(fi os.FileInfo) uint64 {
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		return uint64(st.Dev)
	}
	return 0
}
