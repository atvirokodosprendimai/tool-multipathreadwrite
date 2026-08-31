package iter

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// The whole point: write the set once, then every later call refers to it by
// nothing at all.
func TestRoundTrip(t *testing.T) {
	root := t.TempDir()
	s := Set{Note: "mrw check wiring"}
	if n := s.Add("cmd/mrw/main.go", "internal/check/check.go:1-60", "internal/check/check_test.go"); n != 3 {
		t.Fatalf("Add returned %d, want 3", n)
	}
	if err := Save(root, s); err != nil {
		t.Fatal(err)
	}
	got, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Entries, s.Entries) {
		t.Errorf("entries = %q, want %q", got.Entries, s.Entries)
	}
	if got.Note != "mrw check wiring" {
		t.Errorf("note = %q", got.Note)
	}
}

func TestMissingFileIsAnEmptySetNotAnError(t *testing.T) {
	s, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(s.Entries) != 0 {
		t.Errorf("entries = %q", s.Entries)
	}
}

func TestAddIsIdempotent(t *testing.T) {
	var s Set
	s.Add("a.go", "b.go")
	if n := s.Add("a.go"); n != 0 {
		t.Errorf("re-adding returned %d, want 0", n)
	}
	if len(s.Entries) != 2 {
		t.Errorf("entries = %q", s.Entries)
	}
}

// Removing a bare path drops its ranged entries too: "stop working on this
// file" should not require repeating every range you added for it.
func TestRemoveByPathDropsRangedEntries(t *testing.T) {
	var s Set
	s.Add("a.go:1-10", "a.go:50-60", "b.go")
	if n := s.Remove("a.go"); n != 2 {
		t.Errorf("Remove returned %d, want 2", n)
	}
	if !reflect.DeepEqual(s.Entries, []string{"b.go"}) {
		t.Errorf("entries = %q", s.Entries)
	}
}

func TestPathsStripsRangesAndDeduplicates(t *testing.T) {
	var s Set
	s.Add("a.go:1-10", "a.go:50-60", "b.go", "c.go:/func X/,/^}/")
	want := []string{"a.go", "b.go", "c.go"}
	if got := s.Paths(); !reflect.DeepEqual(got, want) {
		t.Errorf("Paths() = %q, want %q", got, want)
	}
}

func TestLoadIgnoresBlanksAndKeepsTheFirstComment(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".mrw"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "# first note\n\na.go\n# later comment\nb.go\na.go\n"
	if err := os.WriteFile(filepath.Join(root, File), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if s.Note != "first note" {
		t.Errorf("note = %q", s.Note)
	}
	if !reflect.DeepEqual(s.Entries, []string{"a.go", "b.go"}) {
		t.Errorf("entries = %q (duplicates should collapse)", s.Entries)
	}
}

func TestPath(t *testing.T) {
	for spec, want := range map[string]string{
		"a.go":               "a.go",
		"a.go:1-10":          "a.go",
		"a.go:/func X/,/^}/": "a.go",
		"dir/a.go:5":         "dir/a.go",
	} {
		if got := Path(spec); got != want {
			t.Errorf("Path(%q) = %q, want %q", spec, got, want)
		}
	}
}
