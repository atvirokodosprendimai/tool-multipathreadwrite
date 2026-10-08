//go:build windows

package main

import (
	"os"
	"syscall"
)

// toNullDevice reports whether f is the null device (ADR-133): an answer
// written there reached nobody. On Windows a character device carries no file
// identity, so os.SameFile matches NUL against every one of them, a console
// included (the reviews of #350); a handle GetConsoleMode accepts is a console,
// and is not NUL. A stat that fails answers false, so the read records as it
// did before the check existed.
func toNullDevice(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil || fi.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	var mode uint32
	if syscall.GetConsoleMode(syscall.Handle(f.Fd()), &mode) == nil {
		return false
	}
	null, err := os.Stat(os.DevNull)
	return err == nil && os.SameFile(fi, null)
}
