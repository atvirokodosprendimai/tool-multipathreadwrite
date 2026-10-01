//go:build unix

package iter

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// ADR-109 T5, the Codex review of #306. Before ADR-004 the ledger and the
// working set lived in the checkout, under .mrw/, and mrw still reads them
// there: state.Migrate at every CLI start, and the loaders when the state
// directory holds none. Each opened them blocking, so a FIFO at .mrw/seen hung
// every command. None of them waits on one now.
func TestLegacyStateFIFOsAreNotWaitedOn(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, state.LegacyDir), 0o755); err != nil {
		t.Fatal(err)
	}
	pipes := []string{state.LegacyPath(root, seen.Name), state.LegacyPath(root, Name)}
	for _, p := range pipes {
		if err := syscall.Mkfifo(p, 0o600); err != nil {
			t.Skipf("no FIFOs here: %v", err)
		}
	}
	within := func(what string, fn func()) {
		t.Helper()
		done := make(chan struct{})
		go func() { fn(); close(done) }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			for _, p := range pipes {
				if w, err := os.OpenFile(p, os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
					_ = w.Close()
				}
			}
			t.Fatalf("%s blocked on a FIFO in %s", what, state.LegacyDir)
		}
	}
	within("state.Migrate", func() { _, _ = state.Migrate(root) })
	within("seen.Load", func() {
		if l, err := seen.Load(root); err != nil || len(l) != 0 {
			t.Errorf("a FIFO legacy ledger loaded %d record(s), %v", len(l), err)
		}
	})
	within("seen.IsStale", func() { _, _ = seen.IsStale(root) })
	within("iter.Load", func() {
		if s, err := Load(root); err != nil || len(s.Entries) != 0 {
			t.Errorf("a FIFO legacy working set loaded %q, %v", s.Entries, err)
		}
	})
}
