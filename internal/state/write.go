package state

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// renameFn is the seam Write's rename goes through. A rename the platform
// refuses — Windows will not replace a file another process holds open — is
// not one a test can arrange everywhere, so the test refuses it here.
var renameFn = os.Rename

// syncFn is the seam WriteSynced's sync goes through, so a test can see that
// it happens, and happens before the rename.
var syncFn = (*os.File).Sync

// renameTries and renameWait bound how long Write waits out a refused rename:
// on Windows a reader holds the file only while it reads it, so a rename that
// failed is tried again a few times before the write gives up.
const (
	renameTries = 5
	renameWait  = 10 * time.Millisecond
)

// Write replaces name with data (ADR-105): it writes a temp file in the same
// directory and renames it over name, so a reader sees the old file or the new
// one and never part of either, and a run killed mid-write leaves the old file
// whole. os.WriteFile truncated and rewrote in place, and a reader in between
// read a short ledger — a lost licence.
//
// Nothing ever writes name in place. A rename still refused after a few tries
// returns its error with the old file untouched and the temp removed; a file
// whose owner made it read-only is refused, as it was when it was rewritten in
// place, since a rename would replace it whatever its mode (ADR-076 refuses a
// read-only tree file for the same reason). A temp left by a killed run sits
// in the state directory, outside the tree (ADR-004).
func Write(name string, data []byte, perm fs.FileMode) error {
	return write(name, data, perm, false)
}

// WriteSynced is Write with the temp file synced before the rename (ADR-105
// Decision 3), for the files that carry licences: the ledger, the MCP
// acknowledgement store, and a legacy ledger migrated into the state
// directory. A rename is crash-consistent only once the data it publishes is
// on the disk; without the sync a power loss can leave the name pointing at an
// empty file. Measured 2026-09-30 on APFS, where File.Sync is F_FULLFSYNC:
// 4.0 ms median per write, against 0.09 ms unsynced.
func WriteSynced(name string, data []byte, perm fs.FileMode) error {
	return write(name, data, perm, true)
}

func write(name string, data []byte, perm fs.FileMode, sync bool) error {
	if fi, err := os.Stat(name); err == nil && fi.Mode().Perm()&0o200 == 0 {
		return &fs.PathError{Op: "write", Path: name, Err: fmt.Errorf("the file is read-only: %w", fs.ErrPermission)}
	}
	tmp, err := os.CreateTemp(filepath.Dir(name), "."+filepath.Base(name)+".tmp-*")
	if err != nil {
		return err
	}
	_, werr := tmp.Write(data)
	var serr error
	if sync && werr == nil {
		serr = syncFn(tmp)
	}
	if err := errors.Join(werr, serr, tmp.Close()); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := os.Chmod(tmp.Name(), perm); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	for try := 1; ; try++ {
		err = renameFn(tmp.Name(), name)
		if err == nil {
			return nil
		}
		if try == renameTries {
			break
		}
		time.Sleep(time.Duration(try) * renameWait)
	}
	_ = os.Remove(tmp.Name())
	return err
}
