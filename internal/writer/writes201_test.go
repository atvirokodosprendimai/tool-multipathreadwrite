package writer

import (
	"io/fs"
	"os"
	"runtime"
	"syscall"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// ADR-201 T1. A counter that could not be read is not overwritten: the bump
// that follows a read which failed for any reason but "missing" or "not a
// regular file" writes nothing and returns zero. Text that does not parse is
// still read as zero and bumped to 1.
func TestAWriteCounterThatCouldNotBeReadIsNotOverwritten(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	top, _ := state.Checkout(root)
	p, err := state.Path(top, writesName)
	if err != nil {
		t.Fatal(err)
	}
	held := func(t *testing.T, want string) {
		t.Helper()
		if b, err := os.ReadFile(p); err != nil || string(b) != want {
			t.Fatalf("the counter holds %q, %v; want %q", b, err, want)
		}
	}
	if err := os.WriteFile(p, []byte("7\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	real := readCounter
	t.Cleanup(func() { readCounter = real })
	readCounter = func(path string) ([]byte, error) {
		return nil, &fs.PathError{Op: "open", Path: path, Err: syscall.EWOULDBLOCK}
	}
	if got := bumpWrites(root); got != 0 {
		t.Errorf("a bump over a busy counter returned %d, want 0", got)
	}
	held(t, "7\n")
	readCounter = real
	if got := bumpWrites(root); got != 8 {
		t.Errorf("the next bump returned %d, want 8", got)
	}
	held(t, "8\n")

	if runtime.GOOS != "windows" && os.Geteuid() != 0 {
		// Write-only: the read fails and a write would succeed, so only the
		// rule under test keeps the counter (mode 0 is refused by state.Write).
		if err := os.Chmod(p, 0o200); err != nil {
			t.Fatal(err)
		}
		if got := bumpWrites(root); got != 0 {
			t.Errorf("a bump over an unreadable counter returned %d, want 0", got)
		}
		if err := os.Chmod(p, 0o600); err != nil {
			t.Fatal(err)
		}
		held(t, "8\n")
		if got := bumpWrites(root); got != 9 {
			t.Errorf("the bump once it reads again returned %d, want 9", got)
		}
	}

	if err := os.WriteFile(p, []byte("not a number\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := bumpWrites(root); got != 1 {
		t.Errorf("a bump over text that does not parse returned %d, want 1", got)
	}
	held(t, "1\n")
}
