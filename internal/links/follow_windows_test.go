//go:build windows

package links

import (
	"os"
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
	for _, e := range []syscall.Errno{21, 53} { // ERROR_NOT_READY; ERROR_BAD_NETPATH, which Go also reads as ErrNotExist
		if _, err := Through(in, lstatAnswering(e)); err == nil {
			t.Errorf("errno %d ended the walk instead of refusing", e)
		}
	}
}

// ERROR_BAD_NETPATH from Readlink on a junction Lstat described: Go reads it as
// ErrNotExist, which once passed the junction as a reparse point that redirects
// nothing (Codex on #717). It is refused with no path.
func TestAnUnreachableJunctionTargetIsRefused(t *testing.T) {
	r, j := filepath.FromSlash("/r"), filepath.FromSlash("/r/j")
	lfs := FS{
		Lstat: func(p string) (os.FileInfo, error) {
			switch filepath.Clean(p) {
			case r:
				return info{os.ModeDir}, nil
			case j:
				return info{os.ModeIrregular}, nil
			}
			return nil, syscall.Errno(2)
		},
		Readlink: func(string) (string, error) { return "", syscall.Errno(53) },
	}
	if got, err := Through(filepath.FromSlash("/r/j/x"), lfs); err == nil || got != "" {
		t.Errorf("Through = %q, %v, want no path and a refusal", got, err)
	}
}
