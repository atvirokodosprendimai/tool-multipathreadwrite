package bench

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"testing"
)

// TestTheSWEBenchSliceIsTheLowestTwentyHashes holds the selection rule BACKLOG
// registered for Stage 3 before any run: the 20 instances with the lowest
// sha256("adr-156:" + instance_id), whatever order the dataset lists them in.
func TestTheSWEBenchSliceIsTheLowestTwentyHashes(t *testing.T) {
	var ids []string
	for i := range 60 {
		ids = append(ids, fmt.Sprintf("org__repo-%d", 1000+i*7))
	}
	hash := func(id string) string {
		s := sha256.Sum256([]byte("adr-156:" + id))
		return hex.EncodeToString(s[:])
	}
	got := SelectSWEBench(ids, 20)
	if len(got) != 20 {
		t.Fatalf("selected %d, want 20", len(got))
	}
	worst := ""
	for _, id := range got {
		worst = max(worst, hash(id))
	}
	for _, id := range ids {
		if !slices.Contains(got, id) && hash(id) < worst {
			t.Errorf("%s was left out though its hash is lower than a selected one", id)
		}
	}
	rev := slices.Clone(ids)
	slices.Reverse(rev)
	if !slices.Equal(SelectSWEBench(rev, 20), got) {
		t.Error("the selection depends on the dataset's order")
	}
	if n := len(SelectSWEBench(ids[:5], 20)); n != 5 {
		t.Errorf("fewer instances than the slice: selected %d, want all 5", n)
	}
}
