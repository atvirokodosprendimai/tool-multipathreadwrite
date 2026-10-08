//go:build !windows

package main

import "os"

// toNullDevice reports whether f is the null device (ADR-133): an answer
// written there reached nobody. A stat that fails answers false, so the read
// records as it did before the check existed.
func toNullDevice(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil || fi.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	null, err := os.Stat(os.DevNull)
	return err == nil && os.SameFile(fi, null)
}
