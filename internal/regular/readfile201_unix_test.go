//go:build unix

package regular

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// ADR-201 T1. A FIFO is refused at once, by ReadFile and by OpenIn, with an
// error that names the path and wraps ErrNotRegular.
func TestReadFileRefusesANonRegularFileAtOnce(t *testing.T) {
	dir := t.TempDir()
	pipe := filepath.Join(dir, "pipe")
	if err := syscall.Mkfifo(pipe, 0o600); err != nil {
		t.Skipf("no FIFOs here: %v", err)
	}
	t.Cleanup(func() { Release(pipe) })
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Close() })

	for name, call := range map[string]func() error{
		"ReadFile": func() error { _, err := ReadFile(pipe); return err },
		"OpenIn": func() error {
			f, _, err := OpenIn(root, "pipe")
			if f != nil {
				_ = f.Close()
			}
			return err
		},
	} {
		done := make(chan error, 1)
		go func() { done <- call() }()
		select {
		case err := <-done:
			var pe *fs.PathError
			if !errors.Is(err, ErrNotRegular) || !errors.As(err, &pe) || pe.Path == "" {
				t.Errorf("%s on a FIFO: %v, want a *fs.PathError wrapping ErrNotRegular", name, err)
			}
		case <-time.After(5 * time.Second):
			Release(pipe)
			t.Fatalf("%s blocked on a FIFO", name)
		}
	}
}
