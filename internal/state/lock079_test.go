package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ADR-079. Hold is the one lock primitive for every state file: a second
// holder of the same name waits until the first releases, and a release twice
// is harmless.
func TestAStateLockExcludesASecondHolder(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	release, err := Hold(root, "x.lock")
	if err != nil {
		t.Fatal(err)
	}
	got := make(chan func(), 1)
	go func() {
		r, err := Hold(root, "x.lock")
		if err != nil {
			t.Error(err)
		}
		got <- r
	}()
	select {
	case <-got:
		t.Fatal("a second holder took the lock while the first held it")
	case <-time.After(300 * time.Millisecond):
	}
	release()
	release()
	select {
	case r := <-got:
		r()
	case <-time.After(5 * time.Second):
		t.Fatal("the second holder never got the lock after the first released it")
	}
}

// ADR-079, the reviews of #240. Migrate checked that the working set was absent
// and then copied the legacy one, with no lock, so an `iter add` landing between
// the two was overwritten by the older set. It waits for the destination's lock,
// holds it across both, and the live set wins.
func TestMigrateWaitsForTheDestinationsLock(t *testing.T) {
	xdg(t)
	root := t.TempDir()
	legacy := filepath.Join(root, LegacyDir)
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "iteration"), []byte("legacy\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir, err := Dir(root)
	if err != nil {
		t.Fatal(err)
	}
	release, err := Hold(root, "iteration.lock")
	if err != nil {
		t.Fatal(err)
	}
	type result struct {
		moved []string
		err   error
	}
	got := make(chan result, 1)
	go func() {
		m, err := Migrate(root)
		got <- result{m, err}
	}()
	time.Sleep(200 * time.Millisecond)
	live := filepath.Join(dir, "iteration")
	if _, err := os.Stat(live); err == nil {
		release()
		t.Fatal("Migrate copied the legacy set while another holder had its lock")
	}
	// What a locked `iter add` does while it holds the set.
	if err := os.WriteFile(live, []byte("live\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	release()
	var r result
	select {
	case r = <-got:
	case <-time.After(5 * time.Second):
		t.Fatal("Migrate never finished after the lock was released")
	}
	if b, _ := os.ReadFile(live); r.err != nil || string(b) != "live\n" || len(r.moved) != 0 {
		t.Errorf("Migrate over a live set: err %v, moved %v, set %q", r.err, r.moved, b)
	}
}
