package rooted

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ADR-077. With the state base under the root — XDG_STATE_HOME inside the
// checkout, or --root "$HOME" with ~/.local/state — a read served mrw's
// ledger and its ack store, and a plan could edit them. A path inside the base
// is refused before the base exists and after; a sibling of mrw/ is served.
func TestAPathInsideMrwsStateIsRefused(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, ".st"))
	inside := []string{".st/mrw/k/seen", ".st/mrw", ".st/mrw/k/new.txt"}
	for _, p := range inside {
		if _, err := Resolve(root, p); err == nil || !strings.Contains(err.Error(), "own state") {
			t.Errorf("before the base exists, Resolve(%q) = %v, want it refused as mrw's own state", p, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, ".st", "mrw", "k"), 0o700); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{".st/mrw/k/seen": "ledger\n", ".st/notes.txt": "notes\n", "a.txt": "a\n"} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range inside {
		if _, err := Resolve(root, p); err == nil || !strings.Contains(err.Error(), "own state") {
			t.Errorf("Resolve(%q) = %v, want it refused as mrw's own state", p, err)
		}
	}
	for _, p := range []string{".st/notes.txt", ".st", "a.txt"} {
		if _, err := Resolve(root, p); err != nil {
			t.Errorf("Resolve(%q) refused a file that is not mrw's state: %v", p, err)
		}
	}
}

// Codex review of #238. On a filesystem that folds case, `.st/MRW` names the
// base, and the comparison by string let it through. The base is compared as a
// file; on a filesystem that keeps case, `.st/MRW` is another directory and is
// served.
func TestACaseSpellingOfTheStateBaseIsRefused(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, ".st"))
	if err := os.MkdirAll(filepath.Join(root, ".st", "mrw", "k"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".st", "mrw", "k", "pending.json"), []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".st", "MRW")); err != nil {
		if err := os.MkdirAll(filepath.Join(root, ".st", "MRW"), 0o755); err != nil {
			t.Fatal(err)
		}
		if _, err := Resolve(root, ".st/MRW/x.txt"); err != nil {
			t.Errorf("on a filesystem that keeps case, .st/MRW is another directory, and was refused: %v", err)
		}
		return
	}
	if _, err := Resolve(root, ".st/MRW/k/pending.json"); err == nil || !strings.Contains(err.Error(), "own state") {
		t.Errorf("a case spelling of the state base was served: %v", err)
	}
}
