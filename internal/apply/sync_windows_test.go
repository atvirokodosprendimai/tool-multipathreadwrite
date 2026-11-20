//go:build windows

package apply

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"unsafe"
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

// ADR-210 final targeted pass. A path over MAX_PATH reaches CreateFile in the
// extended-length form, and the helper spells drive and share paths as Windows
// wants them.
func TestAWindowsDirectoryPathOver260CharactersIsOpenedInExtendedLengthForm(t *testing.T) {
	for in, want := range map[string]string{
		`C:\work\d`:     `\\?\C:\work\d`,
		`\\srv\share\d`: `\\?\UNC\srv\share\d`,
		`\\?\C:\work\d`: `\\?\C:\work\d`,
		`relative\d`:    `relative\d`,
	} {
		if got := extendedLength(in); got != want {
			t.Errorf("extendedLength(%q) = %q, want %q", in, got, want)
		}
	}
	root := evalOr(t.TempDir())
	tr, err := openTree(root, root)
	if err != nil {
		t.Fatal(err)
	}
	defer tr.close()
	real := createDirFile
	t.Cleanup(func() { createDirFile = real })
	var asked string
	createDirFile = func(name *uint16, a, share uint32, sa *syscall.SecurityAttributes, create, f uint32, tmpl int32) (syscall.Handle, error) {
		asked = syscall.UTF16ToString((*[1 << 16]uint16)(unsafe.Pointer(name))[:])
		return syscall.InvalidHandle, syscall.Errno(2)
	}
	long := root
	for len(long) < 300 {
		long = filepath.Join(long, "dddddddddd")
	}
	_, _ = tr.openDirSync(long)
	if !strings.HasPrefix(asked, `\\?\`) || len(asked) <= 260 {
		t.Errorf("CreateFile was asked for %q (%d), want the extended-length form of a path over 260", asked, len(asked))
	}
}

// ADR-210 final targeted pass. The directory opened by path is compared with the
// one recorded at staging, and a different one (a junction retargeted, here a
// handle for another directory) is refused before anything is flushed.
func TestAWindowsDirectoryOpenedByPathMustBeTheRecordedOne(t *testing.T) {
	root := evalOr(t.TempDir())
	for _, d := range []string{"d1", "d2"} {
		if err := os.Mkdir(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	tr, err := openTree(root, root)
	if err != nil {
		t.Fatal(err)
	}
	defer tr.close()
	rs, rf, rc := SyncsDirs, SyncDirFn, createDirFile
	t.Cleanup(func() { SyncsDirs, SyncDirFn, createDirFile = rs, rf, rc })
	SyncsDirs = true
	flushed := 0
	SyncDirFn = func(*os.File) error { flushed++; return nil }
	recorded, err := tr.stat(filepath.Join(root, "d1"))
	if err != nil {
		t.Fatal(err)
	}
	// Whatever path it is asked for, the "link" leads to d2.
	createDirFile = func(_ *uint16, a, share uint32, sa *syscall.SecurityAttributes, create, f uint32, tmpl int32) (syscall.Handle, error) {
		other, perr := syscall.UTF16PtrFromString(extendedLength(filepath.Join(root, "d2")))
		if perr != nil {
			t.Fatal(perr)
		}
		return rc(other, a, share, sa, create, f, tmpl)
	}
	_, err = syncChanged(tr, []dirRef{{path: filepath.Join(root, "d1"), id: recorded}})
	var se *SyncError
	if !errors.As(err, &se) || flushed != 0 {
		t.Errorf("err=%v, %d flush(es); want a SyncError and no flush when the handle is another directory", err, flushed)
	}
}
