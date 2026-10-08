//go:build windows

package main

import (
	"os"
	"syscall"
	"unsafe"
)

// NtQueryVolumeInformationFile lives in ntdll, not in syscall. LazyDLL keeps
// go.mod at one require (ADR-038).
var (
	modntdll                         = syscall.NewLazyDLL("ntdll.dll")
	procNtQueryVolumeInformationFile = modntdll.NewProc("NtQueryVolumeInformationFile")
)

// fileFsDeviceInformation is the FS_INFORMATION_CLASS that answers a
// FILE_FS_DEVICE_INFORMATION; fileDeviceNull is the device type NUL reports.
const (
	fileFsDeviceInformation = 4
	fileDeviceNull          = 0x15
)

// toNullDevice reports whether f is the null device (ADR-133): an answer
// written there reached nobody. On Windows a character device carries no file
// identity, so os.SameFile matches NUL against every one of them, a console
// included, and GetConsoleMode cannot tell them apart on a handle opened
// without read access (the reviews of #350). The handle's device type names
// NUL itself, whatever access it was opened with. A query that fails answers
// false, so the read records as it did before the check existed.
func toNullDevice(f *os.File) bool {
	if procNtQueryVolumeInformationFile.Find() != nil {
		return false
	}
	var iosb [2]uintptr
	var info struct{ DeviceType, Characteristics uint32 }
	status, _, _ := procNtQueryVolumeInformationFile.Call(f.Fd(), uintptr(unsafe.Pointer(&iosb)),
		uintptr(unsafe.Pointer(&info)), unsafe.Sizeof(info), fileFsDeviceInformation)
	return status == 0 && info.DeviceType == fileDeviceNull
}
