//go:build windows

package apply

import (
	"os"
	"syscall"
	"unsafe"
)

// procSetFileInformationByHandle sets a file's attributes through a handle:
// syscall offers only SetFileAttributes, which goes by name.
var procSetFileInformationByHandle = syscall.NewLazyDLL("kernel32.dll").NewProc("SetFileInformationByHandle")

// fileBasicInfo is FILE_BASIC_INFO. A zero time leaves that time as it is.
type fileBasicInfo struct {
	CreationTime, LastAccessTime, LastWriteTime, ChangeTime int64
	FileAttributes                                          uint32
	_                                                       uint32
}

// fileBasicInfoClass is FileBasicInfo in FILE_INFO_BY_HANDLE_CLASS.
const fileBasicInfoClass = 0

// keepAttributes gives the staged file to the Hidden and System attributes of
// the file at from, which the commit rename replaces (ADR-076). from is read
// through the root and the attributes are set through to's open handle, never
// by name: a name can be redirected by a directory swapped after staging
// (ADR-106). A from that does not exist yet — a create — has nothing to give.
func keepAttributes(tr *tree, from string, to *os.File) error {
	fi, err := tr.stat(from)
	if err != nil {
		return nil
	}
	have, ok := fi.Sys().(*syscall.Win32FileAttributeData)
	if !ok {
		return nil
	}
	const keep = syscall.FILE_ATTRIBUTE_HIDDEN | syscall.FILE_ATTRIBUTE_SYSTEM
	if have.FileAttributes&keep == 0 {
		return nil
	}
	var cur syscall.ByHandleFileInformation
	if err := syscall.GetFileInformationByHandle(syscall.Handle(to.Fd()), &cur); err != nil {
		return err
	}
	info := fileBasicInfo{FileAttributes: cur.FileAttributes | have.FileAttributes&keep}
	if r, _, err := procSetFileInformationByHandle.Call(to.Fd(), fileBasicInfoClass, uintptr(unsafe.Pointer(&info)), unsafe.Sizeof(info)); r == 0 {
		return err
	}
	return nil
}
