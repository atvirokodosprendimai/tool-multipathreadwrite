package rooted

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// fileReadAttributes is the access that reads a file's metadata and nothing
// else.
const fileReadAttributes = 0x80

// identity is the file identity and number of names of the file at path, from
// a handle opened for its attributes only and shared with every other opener.
// os.SameFile opens with share mode 0 and answers false when another process
// holds the file, which would let a ledger held open be served by an alias
// (the Codex review of #361). Windows does not put the count in the FileInfo.
func identity(_ os.FileInfo, path string) (fileKey, uint64, error) {
	p, err := syscall.UTF16PtrFromString(extendedForOpen(path))
	if err != nil {
		return fileKey{}, 0, err
	}
	h, err := syscall.CreateFile(p, fileReadAttributes,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE, nil,
		syscall.OPEN_EXISTING, syscall.FILE_FLAG_BACKUP_SEMANTICS|syscall.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return fileKey{}, 0, err
	}
	defer func() { _ = syscall.CloseHandle(h) }()
	var info syscall.ByHandleFileInformation
	if err := syscall.GetFileInformationByHandle(h, &info); err != nil {
		return fileKey{}, 0, err
	}
	return fileKey{uint64(info.VolumeSerialNumber), uint64(info.FileIndexHigh)<<32 | uint64(info.FileIndexLow)}, uint64(info.NumberOfLinks), nil
}

// extendedForOpen gives an absolute path longer than MAX_PATH its extended
// spelling, which CreateFile needs where os.Open adds it itself.
func extendedForOpen(p string) string {
	if len(p) < 248 || !filepath.IsAbs(p) || strings.HasPrefix(p, `\\?\`) {
		return p
	}
	if strings.HasPrefix(p, `\\`) {
		return `\\?\UNC\` + p[2:]
	}
	return `\\?\` + p
}
