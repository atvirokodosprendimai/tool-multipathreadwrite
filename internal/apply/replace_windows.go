//go:build windows

package apply

import (
	"errors"
	"io/fs"
	"path/filepath"
	"syscall"
	"unsafe"
)

// The Win32 errors a refused open of a target answers that no portable error
// names (ADR-125): another process holds the file without sharing what the
// open asked for, or the name is one the system will not make.
const (
	errorSharingViolation syscall.Errno = 32
	errorInvalidName      syscall.Errno = 123
)

// platformCause names a Windows sharing violation and an invalid name.
func platformCause(err error) string {
	switch {
	case errors.Is(err, errorSharingViolation):
		return "held open by another process"
	case errors.Is(err, errorInvalidName):
		return "not a valid name on this system"
	}
	return ""
}

// The NT calls the probe needs: an open relative to a directory handle, which
// Win32's CreateFile cannot express, and the status-to-error mapping. Reached
// through LazyDLL as internal/state/lock_windows.go reaches kernel32, so
// go.mod keeps one requirement.
var (
	ntdll                    = syscall.NewLazyDLL("ntdll.dll")
	procNtCreateFile         = ntdll.NewProc("NtCreateFile")
	procRtlNtStatusToDosErrN = ntdll.NewProc("RtlNtStatusToDosError")
)

// The access, flags and options the probe opens with: DELETE, the right a
// rename over a file or its removal needs, with every kind of sharing; the
// entry itself, not a link it names; synchronous, as a backup-intent open so a
// directory entry opens too.
const (
	accessDelete              = 0x00010000
	accessSynchronize         = 0x00100000
	objCaseInsensitive        = 0x00000040
	fileOpen                  = 0x00000001
	fileSynchronousIONonAlert = 0x00000020
	fileOpenForBackupIntent   = 0x00004000
	fileOpenReparsePoint      = 0x00200000
)

type unicodeString struct {
	Length        uint16
	MaximumLength uint16
	Buffer        *uint16
}

type objectAttributes struct {
	Length                   uint32
	RootDirectory            syscall.Handle
	ObjectName               *unicodeString
	Attributes               uint32
	SecurityDescriptor       uintptr
	SecurityQualityOfService uintptr
}

type ioStatusBlock struct {
	Status      uintptr
	Information uintptr
}

// replaceable opens full's entry for DELETE with every kind of sharing and
// closes it at once (ADR-125). A holder that did not share delete refuses the
// open with a sharing violation, exactly as it would refuse the commit's
// rename, and nothing has been written yet.
//
// The open is confined as the commit is: the parent directory is opened
// through the root (os.Root, ADR-106), and the leaf relative to that handle,
// so a parent swapped for a junction that leads out of the root is refused
// rather than followed (the Codex review of #338). Relative to a handle, the
// name is one component, so no MAX_PATH limit applies either.
func replaceable(tr *tree, full string) error {
	rel, err := tr.rel(full)
	if err != nil {
		return &fs.PathError{Op: "open", Path: full, Err: err}
	}
	parent, err := tr.r.Open(filepath.Dir(rel))
	if err != nil {
		return err
	}
	defer func() { _ = parent.Close() }()
	name, err := syscall.UTF16FromString(filepath.Base(rel))
	if err != nil {
		return &fs.PathError{Op: "open", Path: full, Err: err}
	}
	n := uint16((len(name) - 1) * 2)
	us := unicodeString{Length: n, MaximumLength: n, Buffer: &name[0]}
	oa := objectAttributes{
		RootDirectory: syscall.Handle(parent.Fd()),
		ObjectName:    &us,
		Attributes:    objCaseInsensitive,
	}
	oa.Length = uint32(unsafe.Sizeof(oa))
	var h syscall.Handle
	var iosb ioStatusBlock
	status, _, _ := procNtCreateFile.Call(
		uintptr(unsafe.Pointer(&h)),
		accessDelete|accessSynchronize,
		uintptr(unsafe.Pointer(&oa)),
		uintptr(unsafe.Pointer(&iosb)),
		0, 0,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
		fileOpen,
		fileSynchronousIONonAlert|fileOpenForBackupIntent|fileOpenReparsePoint,
		0, 0)
	if status != 0 {
		code, _, _ := procRtlNtStatusToDosErrN.Call(status)
		return &fs.PathError{Op: "open", Path: full, Err: syscall.Errno(code)}
	}
	_ = syscall.CloseHandle(h)
	return nil
}
