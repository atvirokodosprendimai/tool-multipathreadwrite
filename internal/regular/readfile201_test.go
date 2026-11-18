package regular

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// ADR-201 T1. ReadFile reads a regular file whole and OpenIn opens one through
// an *os.Root; each error names the path, so statehint.About still recognises a
// state failure by it. Runs on every platform: no FIFO here.
func TestReadFileReadsARegularFileAndNamesWhatItCannot(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(file, []byte("a\nb\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if b, err := ReadFile(file); err != nil || string(b) != "a\nb\n" {
		t.Errorf("a regular file read as %q, %v", b, err)
	}

	missing := filepath.Join(dir, "missing")
	_, err := ReadFile(missing)
	var pe *fs.PathError
	if !errors.Is(err, fs.ErrNotExist) || !errors.As(err, &pe) || pe.Path != missing {
		t.Errorf("a missing file: %v, want a *fs.PathError naming it that is fs.ErrNotExist", err)
	}

	_, err = ReadFile(dir)
	pe = nil
	if err == nil || !errors.As(err, &pe) || pe.Path != dir {
		t.Errorf("a directory: %v, want a *fs.PathError naming it", err)
	}

	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Close() })
	f, fi, err := OpenIn(root, "a.txt")
	if err != nil || !fi.Mode().IsRegular() {
		t.Fatalf("OpenIn of a regular file: %v", err)
	}
	if b, rerr := io.ReadAll(f); rerr != nil || string(b) != "a\nb\n" {
		t.Errorf("OpenIn handed back a file that reads %q, %v", b, rerr)
	}
	_ = f.Close()
	_, _, err = OpenIn(root, "missing")
	pe = nil
	if !errors.Is(err, fs.ErrNotExist) || !errors.As(err, &pe) || pe.Path == "" {
		t.Errorf("OpenIn of a missing file: %v, want a *fs.PathError that is fs.ErrNotExist", err)
	}
}
