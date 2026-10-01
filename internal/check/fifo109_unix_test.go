//go:build unix

package check

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/lines"
)

// ADR-109 T4. Load read .quality-harness.json with os.ReadFile, so a FIFO there
// hung `mrw check` and every write whose check was due. It is refused at once,
// naming the file; a regular config still loads.
func TestAFIFOConfigIsRefusedAtOnce(t *testing.T) {
	root := t.TempDir()
	cfg := filepath.Join(root, ".quality-harness.json")
	if err := syscall.Mkfifo(cfg, 0o600); err != nil {
		t.Skipf("no FIFOs here: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := Load(root)
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), ".quality-harness.json") || !strings.Contains(err.Error(), lines.NotRegular) {
			t.Errorf("a FIFO config was not refused naming it: %v", err)
		}
	case <-time.After(5 * time.Second):
		if w, err := os.OpenFile(cfg, os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
			_ = w.Close()
		}
		t.Fatal("Load blocked on a FIFO config")
	}

	if err := os.Remove(cfg); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte(`{"check":"exit 0"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if c, err := Load(root); err != nil || c.Check != "exit 0" {
		t.Errorf("a regular config did not load: %+v %v", c, err)
	}
}
