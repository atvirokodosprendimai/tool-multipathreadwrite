//go:build windows

package apply

import "syscall"

// keepAttributes gives the staged file at to the Hidden and System attributes
// of the file at from, which the commit rename replaces (ADR-076). A from that
// does not exist yet — a create — has nothing to give.
func keepAttributes(from, to string) error {
	f, err := syscall.UTF16PtrFromString(from)
	if err != nil {
		return err
	}
	have, err := syscall.GetFileAttributes(f)
	if err != nil {
		return nil
	}
	const keep = syscall.FILE_ATTRIBUTE_HIDDEN | syscall.FILE_ATTRIBUTE_SYSTEM
	if have&keep == 0 {
		return nil
	}
	t, err := syscall.UTF16PtrFromString(to)
	if err != nil {
		return err
	}
	cur, err := syscall.GetFileAttributes(t)
	if err != nil {
		return err
	}
	return syscall.SetFileAttributes(t, cur|have&keep)
}
