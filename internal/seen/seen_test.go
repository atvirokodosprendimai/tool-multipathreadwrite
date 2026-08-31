package seen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	root := t.TempDir()
	if err := Record(root, map[string]string{"a.go": "aaa", "b.go": "bbb"}); err != nil {
		t.Fatal(err)
	}
	l, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if l["a.go"] != "aaa" || l["b.go"] != "bbb" {
		t.Errorf("ledger = %v", l)
	}
}

func TestMissingLedgerIsEmptyNotAnError(t *testing.T) {
	l, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(l) != 0 {
		t.Errorf("ledger = %v", l)
	}
}

// One command observing two files must not erase what another observed about a
// third: a read of b.go cannot invalidate an earlier read of a.go.
func TestRecordMergesRatherThanReplaces(t *testing.T) {
	root := t.TempDir()
	if err := Record(root, map[string]string{"a.go": "aaa"}); err != nil {
		t.Fatal(err)
	}
	if err := Record(root, map[string]string{"b.go": "bbb"}); err != nil {
		t.Fatal(err)
	}
	l, _ := Load(root)
	if len(l) != 2 || l["a.go"] != "aaa" {
		t.Errorf("the second record dropped the first: %v", l)
	}
}

func TestRecordOverwritesTheSamePath(t *testing.T) {
	root := t.TempDir()
	_ = Record(root, map[string]string{"a.go": "old"})
	_ = Record(root, map[string]string{"a.go": "new"})
	l, _ := Load(root)
	if l["a.go"] != "new" {
		t.Errorf("a.go = %q, want the newer observation", l["a.go"])
	}
	// And exactly one line, not two.
	b, err := os.ReadFile(filepath.Join(root, File))
	if err != nil {
		t.Fatal(err)
	}
	if n := len(strings.Fields(strings.TrimSpace(string(b)))); n != 2 {
		t.Errorf("ledger holds %d fields, want 2 (one sha + one path):\n%s", n, b)
	}
}

func TestForget(t *testing.T) {
	root := t.TempDir()
	_ = Record(root, map[string]string{"a.go": "aaa", "b.go": "bbb"})
	n, err := Forget(root, []string{"a.go", "never-recorded.go"})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("Forget returned %d, want 1 — an unrecorded path is not a removal", n)
	}
	l, _ := Load(root)
	if _, ok := l["a.go"]; ok {
		t.Error("a.go survived Forget")
	}
	if l["b.go"] != "bbb" {
		t.Error("Forget removed a path it was not asked to")
	}
}

func TestSHAIsStable(t *testing.T) {
	if SHA([]byte("hello")) != SHA([]byte("hello")) {
		t.Error("SHA is not deterministic")
	}
	if SHA([]byte("hello")) == SHA([]byte("hellp")) {
		t.Error("SHA collides on a one-byte difference")
	}
}
