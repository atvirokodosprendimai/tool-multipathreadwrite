//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// ADR-103 T2. A checkout and a junction to it keyed two state directories, so
// two mrw processes held two writer locks for one tree. They key one.
func TestAJunctionSpellingSharesTheCheckoutsState(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	base := t.TempDir()
	target, junction := filepath.Join(base, "repo"), filepath.Join(base, "j")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", junction, target).CombinedOutput(); err != nil {
		t.Fatalf("mklink /J: %v\n%s", err, out)
	}
	a, err := state.Path(target, "seen.write.lock")
	if err != nil {
		t.Fatal(err)
	}
	b, err := state.Path(junction, "seen.write.lock")
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Errorf("one checkout, two writer locks:\n  %s\n  %s", a, b)
	}
}
