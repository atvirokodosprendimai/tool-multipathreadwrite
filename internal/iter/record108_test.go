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

	// The bound is on the line Save writes, "# " included: a note Save accepts
	// round-trips, and one byte more is refused rather than lost on the next load.
	note := strings.Repeat("n", maxEntryBytes-2)
	if err := Save(root, Set{Note: note}); err != nil {
		t.Fatalf("a note whose line fits the bound was refused: %v", err)
	}
	if s, err := Load(root); err != nil || s.Note != note {
		t.Errorf("a note of %d bytes did not round-trip: %d bytes back, %v", len(note), len(s.Note), err)
	}
	if err := Save(root, Set{Note: note + "n"}); err == nil {
		t.Error("a note whose line is one byte over the bound was saved")
	}
	entry := strings.Repeat("e", maxEntryBytes)
	if err := Save(root, Set{Entries: []string{entry}}); err != nil {
		t.Fatalf("an entry of exactly the bound was refused: %v", err)
	}
	if s, err := Load(root); err != nil || !slices.Equal(s.Entries, []string{entry}) {
		t.Errorf("an entry of exactly the bound did not round-trip: %d entries, %v", len(s.Entries), err)
	}

	// Line ends as bufio.ScanLines read them: "\n" and "\r\n" end a line, a final
	// line needs no terminator — even one that fills the reader's buffer
	// exactly — and its trailing "\r" is dropped; edge spaces stay (ADR-069).
	full := strings.Repeat("c", 4096)
	if err := os.WriteFile(p, []byte("a.go\r\n x.go \n"+full), 0o600); err != nil {
		t.Fatal(err)
	}
	if s, err := Load(root); err != nil || !slices.Equal(s.Entries, []string{"a.go", " x.go ", full}) {
		t.Errorf("line ends were not read as ScanLines read them: %d entries %q…, %v", len(s.Entries), s.Entries[:min(2, len(s.Entries))], err)
	}
	if err := os.WriteFile(p, []byte("a.go\nb.go\r"), 0o600); err != nil {
		t.Fatal(err)
	}
	if s, err := Load(root); err != nil || !slices.Equal(s.Entries, []string{"a.go", "b.go"}) {
		t.Errorf("a final line ending in \\r kept it: %q, %v", s.Entries, err)
	}
}
