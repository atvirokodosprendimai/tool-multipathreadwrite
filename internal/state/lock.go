package state

import (
	"os"
	"path/filepath"
	"sync"
)

// Hold opens the lock file name in root's state directory and takes it
// exclusively, waiting while another holder has it, and returns the function
// that releases it; calling that again does nothing. The kernel releases the
// lock if the process dies holding it.
//
// It is here, beside the files it guards, because more than the ledger needs
// it (ADR-079): the working set and the tally were rewritten by truncate-and-
// write with no lock, so a process racing another read an emptied file,
// rebuilt from nothing and saved — wiping the whole tally or working set, not
// one count. A lock file is locked once per process at a time: one process
// opening it a second time waits on itself, so a lock is never taken inside
// another hold of the same name.
func Hold(root, name string) (func(), error) {
	path, err := Path(root, name)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	if err := lock(f); err != nil {
		f.Close()
		return nil, err
	}
	var once sync.Once
	return func() {
		once.Do(func() {
			_ = unlock(f)
			f.Close()
		})
	}, nil
}
