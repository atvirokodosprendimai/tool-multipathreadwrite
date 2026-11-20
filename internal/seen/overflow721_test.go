package seen

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// Issue #721. A ledger record that grew past maxRecordBytes was left out by save,
// and with it every licence the file already held: the merged record replaced the
// previous one in the loaded ledger, and the previous one was never written back.
// The previous record is kept, so earlier licences stand; the line that did not
// fit is not licensed, and a licence-only read that asked is told so.
func TestARecordThatOutgrowsTheBoundKeepsItsEarlierLicences(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	sha := strings.Repeat("a", 64)
	const file = "f.txt"
	var first [][2]int
	for _, n := range []int{1, 3, 5, 7, 9} {
		first = append(first, [2]int{n, n})
	}
	line := fmt.Sprintf("%s  %s  %s\n", sha, formatSpans(Observation{SHA: sha, Spans: first}), file)
	old := maxRecordBytes
	t.Cleanup(func() { maxRecordBytes = old })
	maxRecordBytes = int64(len(line)) + 2 // the five spans fit; a sixth does not

	if err := Record(root, map[string]Observation{file: {SHA: sha, Spans: first}}); err != nil {
		t.Fatal(err)
	}
	more := map[string]Observation{file: {SHA: sha, Spans: [][2]int{{11, 11}}}}
	if err := Record(root, more); err != nil {
		t.Fatalf("Record of an overflowing record returned %v, want nil as on main", err)
	}
	snap, err := Snapshot(root)
	if err != nil {
		t.Fatal(err)
	}
	o, ok := snap[file]
	if !ok || o.SHA != sha {
		t.Fatalf("the record was dropped: %+v", snap)
	}
	for _, n := range []int{1, 3, 5, 7, 9} {
		if !o.Covers(n, n) {
			t.Errorf("line %d lost its licence when line 11 overflowed the record", n)
		}
	}
	if o.Covers(11, 11) || o.Covers(2, 2) {
		t.Errorf("the record licenses a line it was never shown: %+v", o)
	}
	// A licence-only read that asked for line 11 is told the ledger cannot hold it.
	err = Unrecorded(root, more)
	var om *OmittedError
	if !errors.As(err, &om) || !om.Size {
		t.Errorf("Unrecorded returned %v, want a size omission naming the file", err)
	}
	// The pair: asking about what the ledger does hold is not refused.
	if err := Unrecorded(root, map[string]Observation{file: {SHA: sha, Spans: first[:2]}}); err != nil {
		t.Errorf("a licence the ledger holds was refused: %v", err)
	}
}

// A whole-file read cannot be narrowed: advice to read a smaller range would
// send the caller back to the same refusal (the review of #740).
func TestAWholeReadTheLedgerCannotHoldIsNotToldToReadASmallerRange(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	old := maxRecordBytes
	t.Cleanup(func() { maxRecordBytes = old })
	maxRecordBytes = 70
	sha := strings.Repeat("b", 64)
	whole := map[string]Observation{"f.txt": {SHA: sha}}
	if err := Record(root, whole); err != nil {
		t.Fatal(err)
	}
	err := Unrecorded(root, whole)
	var om *OmittedError
	if !errors.As(err, &om) || !om.WholeSize || om.Size {
		t.Fatalf("Unrecorded returned %v, want a whole-file size omission", err)
	}
	if strings.Contains(err.Error(), "smaller range") || !strings.Contains(err.Error(), "shorter") {
		t.Errorf("advice does not fit a whole read: %v", err)
	}
	// The pair: a range that does not fit is still told to read a smaller one.
	err = Unrecorded(root, map[string]Observation{"f.txt": {SHA: sha, Spans: [][2]int{{1, 1}, {3, 3}}}})
	if !errors.As(err, &om) || !om.Size || !strings.Contains(err.Error(), "smaller range") {
		t.Errorf("a range omission lost its advice: %v", err)
	}
}
