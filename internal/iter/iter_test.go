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

func TestResolvePointers(t *testing.T) {
	var s Set
	s.Add("a.go", "b.go:10-20", "c.go")

	for name, tc := range map[string]struct {
		tok  string
		want []string
	}{
		"literal passes through": {"d.go:1-2", []string{"d.go:1-2"}},
		"single":                 {"@2", []string{"b.go:10-20"}},
		"range":                  {"@1-2", []string{"a.go", "b.go:10-20"}},
		"all":                    {"@*", []string{"a.go", "b.go:10-20", "c.go"}},
		"override replaces the entry's own range": {"@2:5", []string{"b.go:5"}},
	} {
		got, err := s.Resolve(tc.tok)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: Resolve(%q) = %q, want %q", name, tc.tok, got, tc.want)
		}
	}
}

// An out-of-range pointer must error, not resolve to nothing: a batch that
// quietly does less than it was asked to is the failure this tool exists for.
func TestOutOfRangePointerIsAnError(t *testing.T) {
	var s Set
	s.Add("a.go")
	for _, tok := range []string{"@2", "@0", "@x", "@1-9", "@3-1"} {
		if got, err := s.Resolve(tok); err == nil {
			t.Errorf("Resolve(%q) = %q, want an error", tok, got)
		}
	}
	var empty Set
	if _, err := empty.Resolve("@*"); err == nil {
		t.Error("@* on an empty set succeeded")
	}
}

// A bare number is a legal filename, so only the sigil may mean "pointer".
func TestBareNumberIsAPathNotAPointer(t *testing.T) {
	var s Set
	s.Add("a.go")
	got, err := s.Resolve("1")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"1"}) {
		t.Errorf("Resolve(\"1\") = %q, want it treated as a path", got)
	}
	if IsPointer("1") || IsPointer("@") || !IsPointer("@1") {
		t.Error("IsPointer disagrees with the sigil rule")
	}
}

func TestResolveAllFlattens(t *testing.T) {
	var s Set
	s.Add("a.go", "b.go", "c.go")
	got, err := s.ResolveAll([]string{"@1-2", "d.go"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"a.go", "b.go", "d.go"}) {
		t.Errorf("ResolveAll = %q", got)
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
