package iter

import (
	"path/filepath"
	"reflect"
	"testing"
)

// Issue #715 (b). A file named `#notes` was added, counted, saved as the line
// `#notes` and loaded back as a comment, so `mrw iter add` said 1 and `mrw iter`
// showed 0. Add keeps the path by writing it `./#notes`, as a --files-from list
// writes a name that would otherwise read as something else.
func TestAFileNamedWithALeadingHashSurvivesSaveAndLoad(t *testing.T) {
	root := t.TempDir()
	var s Set
	if n := s.Add("#notes", "plain.go"); n != 2 {
		t.Fatalf("Add returned %d, want 2", n)
	}
	if err := Save(root, s); err != nil {
		t.Fatal(err)
	}
	got, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Entries, s.Entries) || len(got.Entries) != 2 {
		t.Fatalf("loaded %q, saved %q: the # entry did not come back as an entry", got.Entries, s.Entries)
	}
	if p := filepath.Clean(Path(got.Entries[0])); p != "#notes" {
		t.Errorf("the entry names %q, want the file #notes", p)
	}
	if got.Note != "" {
		t.Errorf("the # entry became the note %q", got.Note)
	}
	// Typed the way the caller typed it, it is the same entry to add and remove.
	if n := got.Add("#notes"); n != 0 {
		t.Errorf("adding #notes again added %d, want 0", n)
	}
	if n := got.Remove("#notes"); n != 1 || len(got.Entries) != 1 {
		t.Errorf("removing #notes removed %d, left %q", n, got.Entries)
	}
	// The pair: a name with a hash inside it was never at risk and is unchanged.
	var t2 Set
	t2.Add("a#b.go")
	if !reflect.DeepEqual(t2.Entries, []string{"a#b.go"}) {
		t.Errorf("a#b.go was rewritten to %q", t2.Entries)
	}
}

// A name whose `#` follows a blank is as much a comment to load as one that
// begins with it, so Add keeps it too.
func TestAnEntryWithABlankBeforeItsHashSurvivesSaveAndLoad(t *testing.T) {
	root := t.TempDir()
	var s Set
	if n := s.Add(" #sp"); n != 1 {
		t.Fatalf("Add returned %d, want 1", n)
	}
	if err := Save(root, s); err != nil {
		t.Fatal(err)
	}
	got, err := Load(root)
	if err != nil || len(got.Entries) != 1 || got.Note != "" {
		t.Errorf("loaded %q note %q (err %v) from %q", got.Entries, got.Note, err, s.Entries)
	}
}
