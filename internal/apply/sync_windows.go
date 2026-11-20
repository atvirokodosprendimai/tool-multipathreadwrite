//go:build windows

package apply

import "os"

// On Windows File.Sync is FlushFileBuffers, which needs a handle opened for
// writing; os.Open of a directory gives none, so no directory is synced and what
// is promised is the staged bytes (ADR-210 Decision 3).
const (
	drains    = false
	syncsDirs = false
)

func platformSyncTemp(f *os.File) error { return f.Sync() }

func platformDrain(*os.File) error { return nil }

func platformSyncDir(f *os.File) error { return f.Sync() }

func platformDevice(os.FileInfo) uint64 { return 0 }
