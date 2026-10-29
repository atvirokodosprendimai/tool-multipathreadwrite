package main

import (
	"strings"
	"testing"
)

// A reference beside a budgeted grep is no clash: a file the caller holds is
// answered by reference, and a read with a stale sha just prints, budget and all.
func TestIfServedBesideAMatchBudget(t *testing.T) {
	root, sha := ifServed179Tree(t)
	out, code := runIn(t, root, "read", "--grep", "l[135]", "--max-matches", "2", "--if-served", "a.txt="+sha, "a.txt")
	if code != 0 || !strings.Contains(out, "-- same as served: lines 1-6 at sha "+sha) {
		t.Errorf("a reference beside --max-matches exited %d:\n%s", code, out)
	}
	out, code = runIn(t, root, "read", "--grep", "l[135]", "--max-matches", "2", "--if-served", "a.txt=deadbeef", "a.txt")
	if code != 0 || strings.Contains(out, "same as served") || !strings.Contains(out, "l2") || !strings.Contains(out, "-- max-matches 2 reached: 1 more") {
		t.Errorf("a stale sha beside --max-matches exited %d, want the budgeted text:\n%s", code, out)
	}
}
