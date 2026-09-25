package iter

import (
	"testing"
)

// ADR-069 T2. The working set trimmed its lines on Load and its arguments on
// Add and Remove, so an entry for "x " came back as x.
func TestTheWorkingSetKeepsAPathsTrailingSpace(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	var s Set
	s.Note = "wip"
	if n := s.Add("x ", "x", "  "); n != 2 {
		t.Errorf("Add(\"x \", \"x\", \"  \") added %d, want 2 (a blank argument is skipped)", n)
	}
	if err := Save(root, s); err != nil {
		t.Fatal(err)
	}
	got, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 2 || got.Entries[0] != "x " || got.Entries[1] != "x" {
		t.Fatalf("Load gave %q, want [\"x \" \"x\"]", got.Entries)
	}
	if got.Note != "wip" {
		t.Errorf("the note did not survive: %q", got.Note)
	}
	if n := got.Remove("x "); n != 1 || len(got.Entries) != 1 || got.Entries[0] != "x" {
		t.Errorf("Remove(\"x \") removed %d and left %q, want 1 and [\"x\"]", n, got.Entries)
	}
}
