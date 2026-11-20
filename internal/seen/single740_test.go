package seen

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// Final review of #740. A request for one line that overflows the record cannot be
// answered by reading a smaller range: the advice is to read the file whole or to
// forget the ledger. A request of several lines still gets the smaller range.
func TestAnOverflowOfAOneLineRequestDoesNotAdviseASmallerRange(t *testing.T) {
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
	maxRecordBytes = int64(len(line)) + 2
	if err := Record(root, map[string]Observation{file: {SHA: sha, Spans: first}}); err != nil {
		t.Fatal(err)
	}
	ask := func(spans [][2]int) string {
		err := Unrecorded(root, map[string]Observation{file: {SHA: sha, Spans: spans}})
		var om *OmittedError
		if !errors.As(err, &om) || !om.Size {
			t.Fatalf("Unrecorded returned %v, want a size omission", err)
		}
		return err.Error()
	}
	one := ask([][2]int{{11, 11}})
	if strings.Contains(one, "smaller range") || !strings.Contains(one, "seen --forget") || !strings.Contains(one, "read the whole file") {
		t.Errorf("a one-line request was told: %s", one)
	}
	if many := ask([][2]int{{11, 11}, {13, 13}}); !strings.Contains(many, "smaller range") {
		t.Errorf("a request of several lines was told: %s", many)
	}
}
