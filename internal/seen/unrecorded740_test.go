package seen

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// Review of #740, P2. Unrecorded asked the held record about every line of every
// span of the read, each ask scanning the record's spans from the start: 100,000
// two-line spans made about 10 billion comparisons for a record that fits. It is
// one sweep now.
func TestCheckingAHundredThousandSpansIsLinear(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	sha := strings.Repeat("a", 64)
	spans := make([][2]int, 0, 200000)
	for i := 0; i < 200000; i++ {
		spans = append(spans, [2]int{3*i + 1, 3*i + 2})
	}
	obs := map[string]Observation{"f.txt": {SHA: sha, Spans: spans}}
	if err := Record(root, obs); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	start := time.Now()
	go func() { done <- Unrecorded(root, obs) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("a record that fits was refused: %v", err)
		}
		t.Logf("checked in %v", time.Since(start))
	case <-time.After(time.Second):
		t.Fatal("Unrecorded took more than 1s for 200,000 spans: it is not linear")
	}
}

// The sweep answers as the per-line check did: a span not in the record is not
// covered, whatever the order the read listed its spans in.
func TestCoversAnswersAsItDidForSpansInAnyOrder(t *testing.T) {
	r := Observation{SHA: "s", Spans: [][2]int{{1, 3}, {5, 9}, {20, 20}}}
	for _, tc := range []struct {
		spans [][2]int
		want  bool
	}{
		{[][2]int{{1, 3}}, true},
		{[][2]int{{6, 8}, {1, 2}, {20, 20}}, true},
		{[][2]int{{3, 5}}, false}, // the gap at 4
		{[][2]int{{1, 3}, {4, 4}}, false},
		{[][2]int{{9, 10}}, false},
		{[][2]int{{0, 1}}, false},
		{[][2]int{{20, 21}}, false},
		{[][2]int{{5, 4}}, true}, // an empty range is covered trivially
	} {
		if got := covers(r, Observation{SHA: "s", Spans: tc.spans}); got != tc.want {
			t.Errorf("covers(%v, %v) = %v, want %v", r.Spans, tc.spans, got, tc.want)
		}
	}
}

// Review of #740, P3. A whole-file licence merged with the lines already shown of
// the same version is the record that goes over the bound, though the path fits:
// the advice says so, and blames the path only when even the whole-file record
// alone is over.
func TestAWholeFileMergeAtTheExactBoundIsNotBlamedOnThePath(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	sha := strings.Repeat("a", 64)
	old := maxRecordBytes
	t.Cleanup(func() { maxRecordBytes = old })
	maxRecordBytes = 77 // "<sha>  1-1  f.txt\n" is exactly 77 bytes; the w-form is 78
	if err := Record(root, map[string]Observation{"f.txt": {SHA: sha, Spans: [][2]int{{1, 1}}}}); err != nil {
		t.Fatal(err)
	}
	whole := map[string]Observation{"f.txt": {SHA: sha, Written: true}}
	if err := Record(root, whole); err != nil {
		t.Fatal(err)
	}
	err := Unrecorded(root, whole)
	var om *OmittedError
	if !errors.As(err, &om) || !om.MergeSize || om.WholeSize {
		t.Fatalf("Unrecorded returned %v, want a merge-size omission and not a path-too-long one", err)
	}
	if strings.Contains(err.Error(), "path itself is too long") || !strings.Contains(err.Error(), "read the whole file again") {
		t.Errorf("the advice blames the path or names no remedy: %v", err)
	}
	// The pair: a path that really is too long for even the whole-file record.
	maxRecordBytes = 70
	err = Unrecorded(root, map[string]Observation{"longer-name.txt": {SHA: sha}})
	if !errors.As(err, &om) || !om.WholeSize || !strings.Contains(err.Error(), "path itself is too long") {
		t.Errorf("a path too long for even the whole-file record: %v", err)
	}
}
