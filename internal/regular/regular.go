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
	"io"
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

// ReadFile reads the regular file at path whole, opening it without blocking
// (ADR-201): a FIFO, a socket or a device is refused with ErrNotRegular and a
// directory with EISDIR, so a state file replaced by one cannot hold the caller.
// Every error is an *fs.PathError naming path, wrapping ErrNotRegular or the
// system's error, so errors.Is(err, fs.ErrNotExist) holds for a missing file and
// statehint.About still recognises a state failure by its path.
func ReadFile(path string) ([]byte, error) {
	f, _, err := Open(path)
	if err != nil {
		return nil, pathError("open", path, err)
	}
	defer func() { _ = f.Close() }()
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, pathError("read", path, err)
	}
	return b, nil
}

// OpenIn is Open through an *os.Root (ADR-201): name is opened without blocking
// beneath root, and a FIFO, a socket or a device is closed and refused with an
// *fs.PathError wrapping ErrNotRegular. A directory is returned, as Open does.
func OpenIn(root *os.Root, name string) (*os.File, fs.FileInfo, error) {
	f, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		if fi, serr := root.Stat(name); serr == nil && !fi.Mode().IsRegular() && !fi.IsDir() {
			return nil, nil, &fs.PathError{Op: "open", Path: name, Err: ErrNotRegular}
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
		return nil, nil, &fs.PathError{Op: "open", Path: name, Err: ErrNotRegular}
	}
	return f, fi, nil
}

// pathError gives err the path it was about, unless it already names one.
func pathError(op, path string, err error) error {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return err
	}
	return &fs.PathError{Op: op, Path: path, Err: err}
}
