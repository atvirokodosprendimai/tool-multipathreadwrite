//go:build windows

package apply

import (
	"errors"
	"path/filepath"
	"syscall"
	"testing"
)

// ADR-210 (Zy, 2026-11-20). A directory flush that answers ERROR_INVALID_FUNCTION
// (a volume that cannot flush a directory, as SMB answers) is "not supported here":
// the write proceeds with a note. Access is denied (5), and any other answer, is an
// error.
func TestAWindowsDirectoryFlushThatIsNotSupportedIsNotAFailure(t *testing.T) {
	if err := flushResult(&pathErr{errInvalidFunction}); !errors.Is(err, ErrNoBarrier) {
		t.Errorf("ERROR_INVALID_FUNCTION gave %v, want ErrNoBarrier", err)
	}
	denied := &pathErr{syscall.Errno(5)}
	if err := flushResult(denied); err == nil || errors.Is(err, ErrNoBarrier) {
		t.Errorf("Access is denied gave %v, want an error", err)
	}
	if err := flushResult(nil); err != nil {
		t.Errorf("a flush that worked gave %v", err)
	}
}

type pathErr struct{ err error }

func (e *pathErr) Error() string { return e.err.Error() }
func (e *pathErr) Unwrap() error { return e.err }

// ADR-210. The handle a directory is flushed through is opened with write access,
// as a directory, and a refusal to open it is the open's error.
func TestAWindowsDirectoryIsOpenedWithWriteAccessForTheFlush(t *testing.T) {
	root := evalOr(t.TempDir())
	tr, err := openTree(root, root)
	if err != nil {
		t.Fatal(err)
	}
	defer tr.close()
	real := createDirFile
	t.Cleanup(func() { createDirFile = real })
	var access, flags uint32
	createDirFile = func(name *uint16, a, share uint32, sa *syscall.SecurityAttributes, create, f uint32, tmpl int32) (syscall.Handle, error) {
		access, flags = a, f
		return syscall.InvalidHandle, syscall.Errno(5)
	}
	if _, err := tr.openDirSync(filepath.Join(root)); !errors.Is(err, syscall.Errno(5)) {
		t.Fatalf("a refused open returned %v, want its error", err)
	}
	if access&fileWriteData == 0 || access&synchronize == 0 || flags&syscall.FILE_FLAG_BACKUP_SEMANTICS == 0 {
		t.Errorf("opened with access %#x and flags %#x, want write access and backup semantics", access, flags)
	}
}
