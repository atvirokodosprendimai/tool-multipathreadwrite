//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// ADR-176: junctions exercise the shared kept-directory lock without the
// privilege a directory symlink may require on Windows.
func TestStateHomesSharingAKeptDirectoryThroughAJunctionUseOneLock(t *testing.T) {
	k := newKeptRig(t)
	if err := os.MkdirAll(k.dir(), 0o700); err != nil {
		t.Fatal(err)
	}
	other := t.TempDir()
	junction := filepath.Join(other, "mrw-refused")
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", junction, k.dir()).CombinedOutput(); err != nil {
		t.Fatalf("mklink /J: %v\n%s", err, out)
	}
	release, err := state.HoldFileWithin(filepath.Join(k.dir(), ".lock"), 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	k.waitsOnLock(other, release)
}
