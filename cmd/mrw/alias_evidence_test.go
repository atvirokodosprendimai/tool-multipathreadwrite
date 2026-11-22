package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

func TestTheReadReferenceAndNullAdvisoryConsultEveryCurrentSpelling(t *testing.T) {
	sha := seen.SHA([]byte(ifServed179Body))
	for _, tc := range []struct {
		name, spelling  string
		second          seen.Observation
		shown, licensed bool
	}{
		{"third spelling", "third", seen.Observation{SHA: sha, Spans: [][2]int{{3, 4}}}, true, true},
		{"partial exact", "first", seen.Observation{SHA: sha, Spans: [][2]int{{3, 4}}}, true, true},
		{"stale exact", "old", seen.Observation{SHA: sha, Spans: [][2]int{{3, 4}}}, true, true},
		{"licensed only", "third", seen.Observation{SHA: sha, Written: true, Spans: [][2]int{{3, 4}}}, false, true},
		{"stale showing", "third", seen.Observation{SHA: seen.SHA(nil), Spans: [][2]int{{3, 4}}}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("XDG_STATE_HOME", t.TempDir())
			root := grepTree(t, map[string]string{"first": ifServed179Body, "distinct": ifServed179Body})
			for _, alias := range []string{"second", "third", "old"} {
				if err := os.Link(filepath.Join(root, "first"), filepath.Join(root, alias)); err != nil {
					t.Skipf("cannot make hard link: %v", err)
				}
			}
			if err := seen.Record(root, seen.Ledger{
				"first": {SHA: sha, Spans: [][2]int{{1, 2}}}, "second": tc.second,
				"old": {SHA: seen.SHA(nil)}, "distinct": {SHA: sha},
			}); err != nil {
				t.Fatal(err)
			}
			// The null advisory reads the old ledger. A normal read below may add to it.
			advice := stillLicensed(root, seen.Ledger{tc.spelling: {SHA: sha, Spans: [][2]int{{1, 4}}}})
			if strings.Contains(advice, "still licensed") != tc.licensed {
				t.Errorf("null advisory %q, want licence %t", advice, tc.licensed)
			}
			out, code := runIn(t, root, "read", "--if-served", tc.spelling+"="+sha[:8], tc.spelling+":1-3")
			if code != 0 || strings.Contains(out, "same as served") != tc.shown || strings.Contains(out, "l4") == tc.shown {
				t.Fatalf("read exited %d, want reference %t: %s", code, tc.shown, out)
			}
		})
	}
}
