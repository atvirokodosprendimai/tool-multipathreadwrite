package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
)

// reportText renders res through report into a file and returns what it wrote.
func reportText(t *testing.T, res apply.Result, quiet bool) string {
	t.Helper()
	f, err := os.Create(filepath.Join(t.TempDir(), "receipt"))
	if err != nil {
		t.Fatal(err)
	}
	report(f, res, quiet)
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// ADR-105 T1. The human receipt prints `left behind: <path>` for each thing a
// run left in the tree, on an applied result and a failed one and under
// --quiet, and nothing when the list is empty; the --json receipt, which embeds
// the engine's Result, carries left_behind.
func TestTheReceiptsNameWhatWasLeftBehind(t *testing.T) {
	left := []string{".mrw-aside-1", filepath.Join("d", ".mrw-2")}
	for _, res := range []apply.Result{
		{Applied: true, LeftBehind: left},
		{Failed: 1, LeftBehind: left, Hunks: []apply.HunkResult{{Path: "a.go", Addr: "1", Op: "replace", Status: apply.StatusFailed, Reason: "r"}}},
	} {
		for _, quiet := range []bool{false, true} {
			out := reportText(t, res, quiet)
			for _, p := range left {
				if !strings.Contains(out, "left behind: "+p+"\n") {
					t.Errorf("applied=%v quiet=%v: no %q line:\n%s", res.Applied, quiet, "left behind: "+p, out)
				}
			}
		}
	}
	if out := reportText(t, apply.Result{Applied: true}, false); strings.Contains(out, "left behind") {
		t.Errorf("a receipt with nothing left names something:\n%s", out)
	}
	b, err := json.Marshal(receipt{Result: apply.Result{Applied: true, LeftBehind: left}})
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		LeftBehind []string `json:"left_behind"`
	}
	if err := json.Unmarshal(b, &doc); err != nil || len(doc.LeftBehind) != 2 || doc.LeftBehind[0] != left[0] {
		t.Errorf("the --json receipt's left_behind is %q (%v):\n%s", doc.LeftBehind, err, b)
	}
}
