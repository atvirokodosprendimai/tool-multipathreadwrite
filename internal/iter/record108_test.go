package iter

import (
	"os"
	"slices"
	"strings"
	"testing"
)

// ADR-108 T9. Save wrote a note or an entry of any length while load's
// bufio.Scanner stopped at ~64 KiB, so a long `mrw iter note` saved and then
// failed every load — every write, and `mrw iter clear` itself. Save refuses a
// record over the bound, naming it; load skips one, as an older binary could
// have written, and keeps the rest.
func TestAWorkingSetRecordTheWriterSavesTheLoaderReads(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	long := strings.Repeat("n", 70000)
	if err := Save(root, Set{Note: long}); err == nil || !strings.Contains(err.Error(), "70000 bytes") {
		t.Errorf("a 70,000-byte note was saved, or refused without its size: %v", err)
	}
	if err := Save(root, Set{Entries: []string{long}}); err == nil {
		t.Error("a 70,000-byte entry was saved")
	}

	if err := Save(root, Set{Note: "ok", Entries: []string{"a.go:1"}}); err != nil {
		t.Fatal(err)
	}
	p, err := ReadPath(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("# ok\na.go:1\n"+long+"\nb.go:2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := Load(root)
	if err != nil || s.Note != "ok" || !slices.Equal(s.Entries, []string{"a.go:1", "b.go:2"}) {
		t.Errorf("a set holding a 70,000-byte line loaded %q %q, %v; want note ok and a.go:1, b.go:2", s.Note, s.Entries, err)
	}
	if _, err := Update(root, func(s *Set) error { s.Entries = nil; return nil }); err != nil {
		t.Errorf("clearing a set that held a 70,000-byte line failed: %v", err)
	}
}
