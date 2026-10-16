package state

import (
	"os"
	"path/filepath"
	"testing"
)

// ADR-161 Decision 14. Prune keeps the caller's own entry, and the caller's
// ledger is its checkout's: a root below a checkout's top names the top's entry
// among its own, so a prune run from a subdirectory cannot remove it.
func TestAPruneFromASubdirectoryKeepsTheCheckoutsEntry(t *testing.T) {
	xdg(t)
	top := resolved(t, t.TempDir())
	sub := filepath.Join(top, "sub")
	if err := os.MkdirAll(filepath.Join(top, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	got, prefix := Checkout(sub)
	if got != top || prefix != "sub" {
		t.Fatalf("Checkout(sub) = %q, %q; want %q, sub", got, prefix, top)
	}
	names, err := selfNames(sub)
	if err != nil {
		t.Fatal(err)
	}
	if !names[key(top)] || !names[key(sub)] {
		t.Errorf("a root below a checkout's top does not name both entries as its own: %v", names)
	}
	// The pair: a root with no .git above names only itself.
	lone := resolved(t, t.TempDir())
	if names, err := selfNames(lone); err != nil || names[key(top)] {
		t.Errorf("a root outside any checkout names another's entry as its own: %v %v", names, err)
	}
}

// The Codex review of ADR-161 T11: a checkout deleted under a caller in its
// subdirectory is no longer found by Checkout, and its entry is still the
// caller's own, because the caller's state directory named it. An outer
// repository that is gone too is another checkout, and is not.
func TestAPruneFromADeletedCheckoutKeepsItsEntry(t *testing.T) {
	xdg(t)
	outer := resolved(t, t.TempDir())
	top := filepath.Join(outer, "nested")
	sub := filepath.Join(top, "sub")
	for _, d := range []string{filepath.Join(outer, ".git"), filepath.Join(top, ".git"), sub} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// The caller used mrw from sub while the checkout was there.
	if _, err := Dir(sub); err != nil {
		t.Fatal(err)
	}
	if _, err := Dir(outer); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(outer); err != nil {
		t.Fatal(err)
	}
	names, err := selfNames(sub)
	if err != nil {
		t.Fatal(err)
	}
	if !names[key(top)] {
		t.Errorf("a prune from a deleted checkout's subdirectory does not name the checkout's entry as its own: %v", names)
	}
	if names[key(outer)] {
		t.Errorf("a prune from a deleted checkout's subdirectory names an outer repository's entry as its own: %v", names)
	}
}
