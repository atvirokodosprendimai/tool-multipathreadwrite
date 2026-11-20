//go:build !darwin && !windows

package apply

import (
	"os"
	"syscall"
)

// Where File.Sync is fsync (Linux and the BSDs) there is no drain: each temp and
// each directory is synced by itself.
const (
	drains    = false
	syncsDirs = true
	// dirOpenFlags open a directory for a sync: only a directory, never blocking.
	dirOpenFlags = syscall.O_DIRECTORY | syscall.O_NONBLOCK
)

func platformSyncTemp(f *os.File) error { return f.Sync() }

func platformDrain(*os.File) error { return nil }

func platformSyncDir(f *os.File) error { return f.Sync() }

func platformDevice(os.FileInfo) uint64 { return 0 }
