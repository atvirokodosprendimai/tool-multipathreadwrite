// Package regular opens a file for reading only when it is a regular file or a
// directory, and asks the open descriptor rather than the path (ADR-109).
//
// A loader that judged a path by Stat and then opened it blocking hung on a
// file swapped for a FIFO in between: opening a FIFO for reading waits for a
// writer. Opening without blocking returns at once, and the descriptor's
// FileInfo describes what was actually opened.
package regular

import (
	"errors"
	"io/fs"
	"os"
	"syscall"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/lines"
)

// ErrNotRegular is Open's refusal of a FIFO, a socket or a device. Its text is
// lines.NotRegular, the refusal every loader has always given.
var ErrNotRegular = errors.New(lines.NotRegular)

// Open opens path for reading without blocking and returns the file with its
// descriptor's FileInfo. A FIFO, a socket or a device is closed and refused
// with ErrNotRegular; a directory is returned, so its caller keeps the error it
// has always given for one. O_NONBLOCK does not change reads of a regular file
// (on Linux an open can fail EWOULDBLOCK on a file under an incompatible lease,
// where a blocking open would wait for the lease to break), and Go ignores it
// on Windows.
//
// A socket cannot be opened at all — the open fails, EOPNOTSUPP on macOS and
// ENXIO on Linux, before there is a descriptor to ask — so a failed open whose
// path exists and is neither a regular file nor a directory is ErrNotRegular
// too (the Codex review of #306).
func Open(path string) (*os.File, fs.FileInfo, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		if fi, serr := os.Stat(path); serr == nil && !fi.Mode().IsRegular() && !fi.IsDir() {
			return nil, nil, ErrNotRegular
		}
		return nil, nil, err
	}
	fi, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, nil, err
	}
	if !fi.Mode().IsRegular() && !fi.IsDir() {
		_ = f.Close()
		return nil, nil, ErrNotRegular
	}
	return f, fi, nil
}
