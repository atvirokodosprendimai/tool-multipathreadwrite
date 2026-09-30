package state

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// renameFn is the seam Write's rename goes through. A rename the platform
// refuses — Windows will not replace a file another process holds open — is
// not one a test can arrange, so the test refuses it here.
var renameFn = os.Rename

// syncFn is the seam WriteSynced's sync goes through, so a test can see that
// it happens, and happens before the rename.
var syncFn = (*os.File).Sync

// Write replaces name with data (ADR-105): it writes a temp file in the same
// directory and renames it over name, so a reader sees the old file or the new
// one and never part of either, and a run killed mid-write leaves the old file
// whole. os.WriteFile truncated and rewrote in place, and a reader in between
// read a short ledger — a lost licence.
//
// When the rename is refused the temp is removed and the file is written in
// place, as it was before, so no platform is worse off; a file whose owner
// made it read-only is written in place too, so it stays refused. A temp left
// by a killed run sits in the state directory, outside the tree (ADR-004).
func Write(name string, data []byte, perm fs.FileMode) error {
	return write(name, data, perm, false)
}

// WriteSynced is Write with the temp file synced before the rename (ADR-105
// Decision 3), for the two files that carry licences: the ledger and the MCP
// acknowledgement store. A rename is crash-consistent only once the data it
// publishes is on the disk; without the sync a power loss can leave the name
// pointing at an empty file. Measured 2026-09-30 on APFS, where File.Sync is
// F_FULLFSYNC: 4.0 ms median per write, against 0.09 ms unsynced.
func WriteSynced(name string, data []byte, perm fs.FileMode) error {
	return write(name, data, perm, true)
}

func write(name string, data []byte, perm fs.FileMode, sync bool) error {
	inPlace := func() error { return os.WriteFile(name, data, perm) }
	// A state file its owner made read-only stays refused, as it was when it
	// was rewritten in place: a rename would replace it regardless of its
	// mode. ADR-076 refuses a read-only tree file for the same reason.
	if fi, err := os.Stat(name); err == nil && fi.Mode().Perm()&0o200 == 0 {
		return inPlace()
	}
	tmp, err := os.CreateTemp(filepath.Dir(name), "."+filepath.Base(name)+".tmp-*")
	if err != nil {
		return inPlace()
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
	if err := renameFn(tmp.Name(), name); err != nil {
		_ = os.Remove(tmp.Name())
		return inPlace()
	}
	return nil
}
