package seen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	root := t.TempDir()
	if err := Record(root, map[string]Observation{"a.go": {SHA: "aaa"}, "b.go": {SHA: "bbb"}}); err != nil {
		t.Fatal(err)
	}
	l, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if l["a.go"].SHA != "aaa" || l["b.go"].SHA != "bbb" {
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
	if err := Record(root, map[string]Observation{"a.go": {SHA: "aaa"}}); err != nil {
		t.Fatal(err)
	}
	if err := Record(root, map[string]Observation{"b.go": {SHA: "bbb"}}); err != nil {
		t.Fatal(err)
	}
	l, _ := Load(root)
	if len(l) != 2 || l["a.go"].SHA != "aaa" {
		t.Errorf("the second record dropped the first: %v", l)
	}
}

func TestRecordOverwritesTheSamePath(t *testing.T) {
	root := t.TempDir()
	_ = Record(root, map[string]Observation{"a.go": {SHA: "old"}})
	_ = Record(root, map[string]Observation{"a.go": {SHA: "new"}})
	l, _ := Load(root)
	if l["a.go"].SHA != "new" {
		t.Errorf("a.go = %+v, want the newer observation", l["a.go"])
	}
	// And exactly one line, not two.
	lp, err := ReadPath(root)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(lp)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(strings.Fields(strings.TrimSpace(string(b)))); n != 3 {
		t.Errorf("ledger holds %d fields, want 3 (sha, spans, path):\n%s", n, b)
	}
}

func TestForget(t *testing.T) {
	root := t.TempDir()
	_ = Record(root, map[string]Observation{"a.go": {SHA: "aaa"}, "b.go": {SHA: "bbb"}})
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
	if l["b.go"].SHA != "bbb" {
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

// TestMain isolates the whole package from the user's real state directory.
// Without it these tests would write into ~/.local/state/mrw — a test suite
// that leaves state on the machine it ran on is its own small version of the
// bug ADR-004 fixes.
func TestMain(m *testing.M) {
	base, err := os.MkdirTemp("", "mrw-state-*")
	if err != nil {
		panic(err)
	}
	os.Setenv("XDG_STATE_HOME", base)
	code := m.Run()
	os.RemoveAll(base)
	os.Exit(code)
}

// The reported bug, as a test: a Record must leave the working tree untouched.
func TestLedgerIsNotWrittenIntoTheRoot(t *testing.T) {
	root := t.TempDir()
	if err := Record(root, map[string]Observation{"a.go": {SHA: "aaa"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".mrw")); err == nil {
		t.Error("Record created .mrw/ in the working tree — the defect ADR-004 fixes")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("the root holds %d entry/entries after Record, want 0", len(entries))
	}
}

// A pre-ADR-004 ledger is still read, so nobody loses data by upgrading.
func TestLegacyInTreeLedgerIsStillRead(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, ".mrw")
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, Name), []byte("abc123  old.go\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	l, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if l["old.go"].SHA != "abc123" {
		t.Errorf("a legacy in-tree ledger was ignored: %v", l)
	}
	// And a subsequent write goes to the state dir, never back into the tree.
	if err := Record(root, map[string]Observation{"new.go": {SHA: "def456"}}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(legacy, Name))
	if strings.Contains(string(b), "new.go") {
		t.Error("Record wrote into the legacy in-tree ledger")
	}
}
