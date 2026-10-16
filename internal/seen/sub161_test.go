package seen

import (
	"reflect"
	"testing"
)

// ADR-161. A sub read licenses the lines it planned and shows none of them:
// the licence is those lines, not the file (Spans), and ServedLine, which
// occurrence=N counts by, sees nothing (Shown). It must survive the ledger
// file, which the write that applies the plan reads in another process, and
// merge with later reads without widening.
func TestAPlanLicenceThatShowsNothingSurvivesTheLedger(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	rec := func(o Observation) Observation {
		t.Helper()
		if err := Record(root, map[string]Observation{"f.txt": o}); err != nil {
			t.Fatal(err)
		}
		l, err := Load(root)
		if err != nil {
			t.Fatal(err)
		}
		return l["f.txt"]
	}
	got := rec(Observation{SHA: "aa", Spans: [][2]int{{2, 2}, {5, 5}}, Written: true, Shown: [][2]int{}})
	if !got.Written || !reflect.DeepEqual(got.Spans, [][2]int{{2, 2}, {5, 5}}) || len(got.Shown) != 0 {
		t.Fatalf("after the ledger file: %+v, want written, spans 2 and 5, nothing shown", got)
	}
	if !got.Covers(2, 2) || got.Covers(3, 3) || got.Whole() {
		t.Errorf("licence is not exactly lines 2 and 5: %+v", got)
	}
	if got.ServedLine(2) {
		t.Errorf("a planned line counts as shown")
	}
	// A later read of line 3 shows it and licenses it; the licence stays partial.
	got = rec(Observation{SHA: "aa", Spans: [][2]int{{3, 3}}})
	if !got.Written || !reflect.DeepEqual(got.Spans, [][2]int{{2, 3}, {5, 5}}) || !reflect.DeepEqual(got.Shown, [][2]int{{3, 3}}) || got.Covers(4, 4) {
		t.Errorf("after reading line 3: %+v, want spans 2-3 and 5, shown 3", got)
	}
	// A second sub plan over the same version adds lines and keeps what was shown.
	got = rec(Observation{SHA: "aa", Spans: [][2]int{{7, 7}}, Written: true, Shown: [][2]int{}})
	if !reflect.DeepEqual(got.Spans, [][2]int{{2, 3}, {5, 5}, {7, 7}}) || !reflect.DeepEqual(got.Shown, [][2]int{{3, 3}}) {
		t.Errorf("after a second plan: %+v", got)
	}
	// A plan never narrows a version already read whole, and a whole read widens.
	rec(Observation{SHA: "bb"})
	got = rec(Observation{SHA: "bb", Spans: [][2]int{{1, 1}}, Written: true, Shown: [][2]int{}})
	if !got.Whole() || got.Written {
		t.Errorf("a plan narrowed a whole read: %+v", got)
	}
	rec(Observation{SHA: "cc", Spans: [][2]int{{1, 1}}, Written: true, Shown: [][2]int{}})
	got = rec(Observation{SHA: "cc"})
	if !got.Whole() {
		t.Errorf("a whole read did not widen the plan's licence: %+v", got)
	}
}
