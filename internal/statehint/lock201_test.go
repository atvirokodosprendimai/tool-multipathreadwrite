package statehint

import (
	"io/fs"
	"path/filepath"
	"syscall"
	"testing"
)

// ADR-201 Decision 4. A lock that could not be taken names its lock file, which
// is under the state home, and is still not "a state directory that cannot be
// written": taken for one, a read with an unlockable ledger lock would serve
// and exit 0 (ADR-166) where it has always exited 2, under a hint to fix a
// directory that is fine. The same path with another operation is one.
func TestALockFailureIsNotAStateDirectoryFailure(t *testing.T) {
	base := filepath.Join(t.TempDir(), "state", "mrw")
	p := filepath.Join(base, "0123456789abcdef", "seen.lock")
	if about(&fs.PathError{Op: "lock", Path: p, Err: syscall.ENOLCK}, base) {
		t.Error("a lock error under the state home was taken for a state directory failure")
	}
	if !about(&fs.PathError{Op: "open", Path: p, Err: syscall.EACCES}, base) {
		t.Error("an open error under the state home is a state directory failure, and was not taken for one")
	}
}
