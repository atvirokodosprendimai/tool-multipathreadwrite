package plan

import (
	"strings"
	"testing"
)

// ADR-078 T1. Every line under a header that did not parse was reported as
// "text before the first @@ header", burying the one real error; the plan is
// refused with that one error. Stray text before any header is still reported.
func TestABadHeaderIsOneErrorNotOnePerBodyLine(t *testing.T) {
	_, err := Parse(strings.NewReader("@@ a.go 3 replac\nX\nY\n@@ b.go 1 replace\nZ\n"))
	if err == nil || !strings.Contains(err.Error(), "plan has 1 error(s)") || strings.Contains(err.Error(), "text before") {
		t.Errorf("want one error for the bad op: %v", err)
	}
	if _, err := Parse(strings.NewReader("stray\n@@ a.go 1 replace\nX\n")); err == nil || !strings.Contains(err.Error(), "text before the first @@ header") {
		t.Errorf("stray text before the first header is no longer reported: %v", err)
	}
}

// ADR-078 T1. A header written with a tab after @@ read as prose, refused as
// "text before the first @@ header" on the line meant to be one. The tab is
// named, once.
func TestATabAfterTheAtSignsIsNamed(t *testing.T) {
	_, err := Parse(strings.NewReader("@@\ta.go\t1\treplace\nX\n"))
	if err == nil || !strings.Contains(err.Error(), "tab after @@") || !strings.Contains(err.Error(), "plan has 1 error(s)") {
		t.Errorf("want the tab named, as one error: %v", err)
	}
	// Under a header that did not parse, a tab header was swallowed as its
	// body (the review of #239): each is its own error.
	for _, p := range []string{"@@ a.go 1 replac\nX\n@@\ta.go\t2\treplace\nY\n", "@@\ta.go\t1\treplace\nX\n@@\ta.go\t2\treplace\nY\n"} {
		_, err := Parse(strings.NewReader(p))
		if err == nil || !strings.Contains(err.Error(), "line 3: a header is") || !strings.Contains(err.Error(), "plan has 2 error(s)") {
			t.Errorf("%q: want the second header's tab named too: %v", p, err)
		}
	}
}
