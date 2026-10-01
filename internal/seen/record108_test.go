package seen

import (
	"os"
	"strings"
	"testing"
)

// ADR-108 T5. The ledger's loader kept bufio.Scanner's default ~64 KiB line
// limit while the writer put every span of a path on one line: enough sparse
// reads made a record the loader could not read, and since Record loads first,
// no later read could repair it. Save and load share a record bound: a record
// within it reads back, a longer one is left out on save (that file needs
// reading again; nothing is widened), and a ledger line past it — written by an
// older binary — discards the ledger as a stale header does.
func TestALedgerRecordTheWriterSavesTheLoaderReads(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	sha := strings.Repeat("a", 64)
	spans := make([][2]int, 0, 10000)
	for i := 0; i < 10000; i++ {
		spans = append(spans, [2]int{2*i + 1, 2*i + 1})
	}

	root := t.TempDir()
	if err := Record(root, map[string]Observation{"a.txt": {SHA: sha, Spans: spans}}); err != nil {
		t.Fatal(err)
	}
	l, err := Load(root)
	if err != nil || len(l["a.txt"].Spans) != 10000 {
		t.Fatalf("a record of 10,000 spans did not read back: %d spans, %v", len(l["a.txt"].Spans), err)
	}
	if err := Record(root, map[string]Observation{"b.txt": {SHA: sha}}); err != nil {
		t.Fatalf("a Record after the long record failed: %v", err)
	}

	old := maxRecordBytes
	t.Cleanup(func() { maxRecordBytes = old })
	maxRecordBytes = 1000
	small := t.TempDir()
	// b.txt first: a save that wrote a.txt's long record would make the next
	// Load discard the whole ledger, b.txt with it. Recorded the other way
	// round, the Record of b.txt would start from that empty ledger and hide it.
	if err := Record(small, map[string]Observation{"b.txt": {SHA: sha}}); err != nil {
		t.Fatal(err)
	}
	if err := Record(small, map[string]Observation{"a.txt": {SHA: sha, Spans: spans}}); err != nil {
		t.Fatalf("a Record of a record over the bound failed: %v", err)
	}
	l, err = Load(small)
	if err != nil {
		t.Fatal(err)
	}
	if _, kept := l["a.txt"]; kept {
		t.Error("a record over the bound was saved")
	}
	if _, ok := l["b.txt"]; !ok {
		t.Error("a record within the bound was lost")
	}

	p, err := writePath(small)
	if err != nil {
		t.Fatal(err)
	}
	long := header + "\n" + sha + "  " + strings.Repeat("1-1,", 1000) + "1-1  d.txt\n"
	if err := os.WriteFile(p, []byte(long), 0o600); err != nil {
		t.Fatal(err)
	}
	if l, err := Load(small); err != nil || len(l) != 0 {
		t.Errorf("a ledger line past the bound loaded %d record(s), %v; want an empty ledger and no error", len(l), err)
	}
}
