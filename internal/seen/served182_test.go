package seen

import (
	"strings"
	"testing"
)

// ADR-182 Decision 1. A written-shaped observation is described by what was
// shown and what is licensed, and by no source: the ledger line carries none.
func TestServedSaysWhatWasShownAndWhatIsLicensedForEveryShape(t *testing.T) {
	two3 := [][2]int{{2, 3}}
	one3 := [][2]int{{1, 3}}
	none := [][2]int{}
	_, disjoint, ok := parseLine("sha  w2-3|7-8  d.txt")
	if !ok {
		t.Fatal("w2-3|7-8 did not parse")
	}
	for _, c := range []struct {
		name string
		o    Observation
		want string
	}{
		{"whole file licensed, nothing shown", Observation{Written: true, Shown: none}, "no lines, and licensed the whole file"},
		{"whole file licensed, lines shown", Observation{Written: true, Shown: two3}, "lines 2-3, and licensed the whole file"},
		{"lines licensed, nothing shown", Observation{Written: true, Shown: none, Spans: one3}, "no lines, and licensed lines 1-3"},
		{"lines licensed, some shown", Observation{Written: true, Shown: two3, Spans: one3}, "lines 2-3, and licensed lines 1-3"},
		{"empty licence, nothing shown", Observation{Written: true, Shown: none, Spans: none}, "no lines, and licensed none"},
		{"empty licence, lines shown", Observation{Written: true, Shown: two3, Spans: none}, "lines 2-3, and licensed none"},
		{"a disjoint licence is printed as stored", disjoint, "lines 2-3, and licensed lines 7-8"},
		{"read: whole", Observation{}, "the whole file"},
		{"read: none", Observation{Spans: none}, "no lines"},
		{"read: spans", Observation{Spans: [][2]int{{1, 3}, {7, 7}}}, "lines 1-3,7"},
	} {
		got := c.o.Served()
		if got != c.want {
			t.Errorf("%s: Served() = %q, want %q", c.name, got, c.want)
		}
		if strings.Contains(got, "mrw wrote") || strings.Contains(got, "plan") {
			t.Errorf("%s: %q names a source the ledger does not hold", c.name, got)
		}
	}
}
