package apply

import (
	"errors"
	"os"
	"syscall"
	"testing"
)

// ADR-210 review. A signal that interrupts fsync is retried, not reported as a
// failure to sync; any other error is returned at once.
func TestAnInterruptedFsyncIsRetried(t *testing.T) {
	calls := 0
	err := retryEINTR(func() error {
		calls++
		if calls < 3 {
			return syscall.EINTR
		}
		return nil
	})
	if err != nil || calls != 3 {
		t.Errorf("err=%v after %d call(s); want nil after 3", err, calls)
	}
	calls = 0
	boom := errors.New("EIO")
	if err := retryEINTR(func() error { calls++; return boom }); err != boom || calls != 1 {
		t.Errorf("err=%v after %d call(s); want the error at once", err, calls)
	}
}

// ADR-210 (Zy, 2026-11-20). A volume that answers F_FULLFSYNC with ENOTSUP gives no
// device barrier: the drain falls back to a plain fsync of the directory and, when
// that works, answers ErrNoBarrier so the write proceeds and says so. Any other
// error, the fallback's included, is an error; a signal that interrupts the drain
// is retried.
func TestAnUnsupportedDrainFallsBackToFsyncAndSaysSo(t *testing.T) {
	f, err := os.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	real := fullFsync
	t.Cleanup(func() { fullFsync = real })
	fullFsync = func(uintptr) error { return syscall.ENOTSUP }
	if err := platformDrain(f); !errors.Is(err, ErrNoBarrier) {
		t.Errorf("a drain answered ENOTSUP returned %v, want ErrNoBarrier", err)
	}
	eio := errors.New("EIO")
	fullFsync = func(uintptr) error { return eio }
	if err := platformDrain(f); !errors.Is(err, eio) {
		t.Errorf("a drain answered EIO returned %v, want that error", err)
	}
	// The fallback fails too: the directory handle is closed, so fsync errors.
	fullFsync = func(uintptr) error { return syscall.ENOTSUP }
	closed, _ := os.Open(t.TempDir())
	_ = closed.Close()
	if err := platformDrain(closed); err == nil || errors.Is(err, ErrNoBarrier) {
		t.Errorf("an unsupported drain whose fallback failed returned %v, want an error", err)
	}
	calls := 0
	fullFsync = func(uintptr) error {
		calls++
		if calls < 3 {
			return syscall.EINTR
		}
		return nil
	}
	if err := platformDrain(f); err != nil || calls != 3 {
		t.Errorf("an interrupted drain: err=%v after %d call(s), want nil after 3", err, calls)
	}
}
