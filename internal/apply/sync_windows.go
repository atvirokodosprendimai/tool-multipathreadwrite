//go:build windows

package apply

import (
	"errors"
	"os"
	"strings"
	"syscall"
)

// On Windows File.Sync is FlushFileBuffers. os.Open of a directory gives a handle
// that FlushFileBuffers refuses (Access is denied, 5), but a handle opened with
// write access does flush on NTFS, as a Windows peer measured on 2026-11-20
// (ADR-210 Decision 3's amendment), so directories are synced here too, through a
// handle made by openDirSync. There is no drain.
const (
	drains       = false
	syncsDirs    = true
	dirOpenFlags = 0
)

// The access mask of a directory handle that can be flushed: FILE_WRITE_DATA (the
// right to add an entry) and SYNCHRONIZE.
const (
	fileWriteData = 0x2
	synchronize   = 0x100000
	// errInvalidFunction is ERROR_INVALID_FUNCTION, which a directory flush over
	// SMB answers: the volume does not support it.
	errInvalidFunction = syscall.Errno(1)
)

func platformSyncTemp(f *os.File) error { return f.Sync() }

func platformDrain(*os.File) error { return nil }

// platformSyncDir flushes an open directory. ERROR_INVALID_FUNCTION is "not
// supported here" (ErrNoBarrier): the write proceeds and the receipt says so.
// ERROR_ACCESS_DENIED (5) is NOT: with the write access openDirSync asked for, it
// means the directory refuses a flush the user is entitled to, an error that the
// peer's measurements show only for handles opened without write access.
func platformSyncDir(f *os.File) error { return flushResult(f.Sync()) }

// flushResult reads what a directory flush answered: nil is stable, "not
// supported" is ErrNoBarrier, anything else (Access is denied included) an error.
func flushResult(err error) error {
	if errors.Is(err, errInvalidFunction) {
		return ErrNoBarrier
	}
	return err
}

func platformDevice(os.FileInfo) uint64 { return 0 }

// createDirFile is syscall.CreateFile, a variable so a test can answer for it.
var createDirFile = syscall.CreateFile

// openDirSync opens the recorded directory for a flush: through its resolved
// absolute path, with write access, as a directory (FILE_FLAG_BACKUP_SEMANTICS).
// The held root cannot ask for that access on a directory, so the path is judged
// by the root first (inside it, no .git) and the handle's identity is compared by
// the caller with the one recorded at staging.
func (t *tree) openDirSync(p string) (*os.File, error) {
	if _, err := t.rel(p); err != nil {
		return nil, err
	}
	name, err := syscall.UTF16PtrFromString(extendedLength(p))
	if err != nil {
		return nil, err
	}
	h, err := createDirFile(name, fileWriteData|synchronize,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
		nil, syscall.OPEN_EXISTING, syscall.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: p, Err: err}
	}
	return os.NewFile(uintptr(h), p), nil
}

// extendedLength spells an absolute path in the form that lifts MAX_PATH:
// \\?\C:\dir for a drive path and \\?\UNC\server\share\dir for a share. A path
// already in that form, or not absolute, is returned as it is.
func extendedLength(p string) string {
	switch {
	case strings.HasPrefix(p, `\\?\`) || strings.HasPrefix(p, `\\.\`):
		return p
	case strings.HasPrefix(p, `\\`):
		return `\\?\UNC\` + p[2:]
	case len(p) >= 3 && p[1] == ':' && (p[2] == '\\' || p[2] == '/'):
		return `\\?\` + strings.ReplaceAll(p, "/", `\`)
	}
	return p
}
