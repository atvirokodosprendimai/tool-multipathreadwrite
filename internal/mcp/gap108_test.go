package mcp

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"testing"
)

// servedText renders numbered lines the way a read serves them.
func servedText(nums []int) string {
	var b strings.Builder
	b.WriteString("==> a.txt  500L  1000B  sha 0123abcd\n")
	for _, n := range nums {
		fmt.Fprintf(&b, "%6d| line %d\n", n, n)
	}
	return b.String()
}

// licensedLines is every line number the spans cover.
func licensedLines(spans map[string][2]int) map[int]bool {
	out := map[int]bool{}
	for _, s := range spans {
		for n := s[0]; n <= s[1]; n++ {
			out[n] = true
		}
	}
	return out
}

// ADR-108 T1, the record's Enforced-by. interleave grouped served lines into
// checkpoints by count with no break at a gap, and recorded each group as its
// first..last number, so a read serving lines 1 and 100 licensed 1-100 once
// acknowledged. A checkpoint now covers one run of consecutive lines: across
// random served sets, the lines the checkpoints cover are exactly the lines
// served.
func TestACheckpointCoversOnlyConsecutiveServedLines(t *testing.T) {
	_, spans := interleave(servedText([]int{1, 100}))
	if got := licensedLines(spans); len(got) != 2 || !got[1] || !got[100] {
		t.Fatalf("lines 1 and 100 served together license %d line(s): %v", len(got), spans)
	}
	if _, spans := interleave(servedText([]int{7, 8, 9})); len(spans) != 1 {
		t.Errorf("a run of three consecutive lines gave %d checkpoints, want 1: %v", len(spans), spans)
	}
	rng := rand.New(rand.NewSource(108))
	for round := 0; round < 200; round++ {
		set := map[int]bool{}
		for i := rng.Intn(300); i >= 0; i-- {
			set[1+rng.Intn(500)] = true
		}
		var nums []int
		for n := range set {
			nums = append(nums, n)
		}
		sort.Ints(nums)
		_, spans := interleave(servedText(nums))
		got := licensedLines(spans)
		if len(got) != len(set) {
			t.Fatalf("round %d: %d lines served, %d licensed", round, len(set), len(got))
		}
		for n := range got {
			if !set[n] {
				t.Fatalf("round %d: line %d licensed but never served", round, n)
			}
		}
	}
}
