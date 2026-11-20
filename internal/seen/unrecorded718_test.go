package seen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestUnrecordedAsksOnlyWhenSomethingWasLicensedAndNamesWhyItIsMissing: the
// final review of #718 found three edges. A read that licensed nothing is not
// refused, not even when the ledger cannot be read; a newer mrw's ledger, which
// is left as it is, refuses a read that licensed lines; and a record another
// command replaced is named as changed, not as too large.
func TestUnrecordedAsksOnlyWhenSomethingWasLicensedAndNamesWhyItIsMissing(t *testing.T) {
	a, b := strings.Repeat("a", 64), strings.Repeat("b", 64)
	t.Run("nothing licensed, ledger unreadable", func(t *testing.T) {
		state := filepath.Join(t.TempDir(), "state")
		if err := os.WriteFile(state, []byte("a file, not a directory"), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Setenv("XDG_STATE_HOME", state)
		if err := Unrecorded(t.TempDir(), map[string]Observation{"f.txt": {SHA: a, Spans: [][2]int{}}}); err != nil {
			t.Errorf("a read that licensed nothing was refused: %v", err)
		}
	})
	t.Run("a newer ledger", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		plant(t, root, "#mrw-seen v7 v9.9.9 (fffffff)\n"+b+"  -  a.go\n")
		err := Unrecorded(root, map[string]Observation{"f.txt": {SHA: a}})
		if err == nil || !strings.Contains(err.Error(), "newer mrw") {
			t.Errorf("a licence a newer ledger kept out was not refused: %v", err)
		}
	})
	t.Run("a record another command replaced", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("XDG_STATE_HOME", t.TempDir())
		if err := Record(root, map[string]Observation{"f.txt": {SHA: b}}); err != nil {
			t.Fatal(err)
		}
		err := Unrecorded(root, map[string]Observation{"f.txt": {SHA: a}})
		if err == nil || !strings.Contains(err.Error(), "another version") || strings.Contains(err.Error(), "bytes)") {
			t.Errorf("a replaced record was not named as changed: %v", err)
		}
	})
}
