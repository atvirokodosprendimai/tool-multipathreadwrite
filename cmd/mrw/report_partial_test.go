package main

import (
	"os"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
)

// ADR-066 T3. A plan that wrote some files and then failed a commit step is
// neither "applied" nor "NOTHING WRITTEN": the summary line every caller reads
// says PARTIALLY APPLIED. The partial case has to be tested before the
// nothing-written one, or the fix swaps one false word for its opposite.
func TestAPartialCommitIsNotSummarisedAsApplied(t *testing.T) {
	res := apply.Result{
		Hunks: []apply.HunkResult{
			{Path: "a.go", Op: "replace", Status: apply.StatusOK},
			{Path: "b.go", Op: "replace", Status: apply.StatusFailed, Reason: "injected"},
		},
		Files: []apply.FileResult{
			{Path: "a.go", Written: true},
			{Path: "b.go"},
		},
		Failed: 1,
	}
	f, err := os.CreateTemp(t.TempDir(), "report")
	if err != nil {
		t.Fatal(err)
	}
	report(f, res, false)
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	out := string(b)
	if !strings.Contains(out, "PARTIALLY APPLIED") {
		t.Errorf("a partial commit's summary does not say PARTIALLY APPLIED:\n%s", out)
	}
	if strings.Contains(out, "— applied") || strings.Contains(out, "NOTHING WRITTEN") {
		t.Errorf("a partial commit's summary claims applied or nothing written:\n%s", out)
	}
}
