package read

import (
	"reflect"
	"strings"
	"testing"
)

// ADR-137. A read that names no range starts as a whole-file observation, which
// licenses every line. A line cut by --max-cols must demote it: the cut line is
// not recorded, the lines either side are (caught by a surviving mutant that
// left the whole-file licence standing).
func TestAWholeFileReadThatCutALineStillLeavesItOutOfTheLedger(t *testing.T) {
	root := longLineFixture(t)
	sp, err := ParseSpec("a.txt")
	if err != nil {
		t.Fatal(err)
	}
	var sb strings.Builder
	observed, _ := Run(&sb, root, []Spec{sp}, Options{Numbers: true, MaxCols: 40})
	if !strings.Contains(sb.String(), "[cols 1-40 of 400]") {
		t.Fatalf("the long line was not cut from its start:\n%s", sb.String())
	}
	if got, want := observed["a.txt"].Spans, [][2]int{{1, 1}, {3, 3}}; !reflect.DeepEqual(got, want) {
		t.Errorf("a whole-file read that cut line 2 recorded %v, want %v (nil would be the whole file)", got, want)
	}
}
