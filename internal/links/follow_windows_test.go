//go:build windows

package links

import (
	"path/filepath"
	"syscall"
	"testing"
)

// P3-6, on Windows through the walk itself: an Lstat that answers one of the
// invalid-name errnos ends the walk, and ERROR_NOT_READY refuses it.
func TestTheWindowsLstatErrnosThatEndTheWalkAreTheInvalidNames(t *testing.T) {
	in := filepath.FromSlash("/r/x/y")
	for _, e := range []syscall.Errno{123, 161, 206, 267} {
		if got, err := Through(in, lstatAnswering(e)); err != nil || got != in {
			t.Errorf("errno %d: Through = %s, %v, want the path kept and no error", e, got, err)
		}
	}
	if _, err := Through(in, lstatAnswering(syscall.Errno(21))); err == nil {
		t.Error("ERROR_NOT_READY (21) ended the walk instead of refusing")
	}
}
