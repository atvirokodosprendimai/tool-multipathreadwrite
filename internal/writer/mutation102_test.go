package writer

import (
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
)

// ADR-102 T1. Every tally and receipt asks one question — how much of this plan
// reached the tree — and asking Applied alone answered "nothing" for a commit
// that failed after a file had landed (ADR-066).
func TestMutationSaysHowMuchOfAPlanLanded(t *testing.T) {
	written := apply.FileResult{Path: "a.txt", Written: true}
	unwritten := apply.FileResult{Path: "b.txt"}
	for _, c := range []struct {
		name string
		res  apply.Result
		want Mutation
	}{
		{"nothing written", apply.Result{Files: []apply.FileResult{unwritten}}, None},
		{"no files at all", apply.Result{}, None},
		{"a dry run", apply.Result{DryRun: true, Applied: true, Files: []apply.FileResult{written}}, None},
		{"a commit that failed after a file landed", apply.Result{Files: []apply.FileResult{written, unwritten}}, Partial},
		{"a plan that applied", apply.Result{Applied: true, Files: []apply.FileResult{written}}, Complete},
	} {
		if got := MutationOf(c.res); got != c.want {
			t.Errorf("%s: MutationOf = %v, want %v", c.name, got, c.want)
		}
	}
}
