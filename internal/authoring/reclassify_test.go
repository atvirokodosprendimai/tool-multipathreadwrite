package authoring

import "testing"

// ADR-072 T2. The write path counts a landing as applied BEFORE its check runs,
// so a write killed during the check is still counted, and then moves that one
// count to the check's verdict. A move never adds a plan: from is decremented,
// at a floor of zero, as to is incremented.
func TestReclassifyMovesOneCountAndNeverAddsOne(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if err := Record(root, Applied); err != nil {
		t.Fatal(err)
	}
	if err := Reclassify(root, Applied, FailedCheck); err != nil {
		t.Fatal(err)
	}
	tl, _ := Load(root)
	if tl["applied"] != 0 || tl["failed_check"] != 1 || tl.Plans() != 1 {
		t.Fatalf("after one landing and one move: %v (plans %d), want applied 0, failed_check 1, one plan", tl, tl.Plans())
	}
	// Nothing left in applied: the move still records the verdict and never
	// takes applied below zero.
	if err := Reclassify(root, Applied, CheckNotRun); err != nil {
		t.Fatal(err)
	}
	tl, _ = Load(root)
	if tl["applied"] != 0 || tl["check_not_run"] != 1 {
		t.Fatalf("a move with nothing to move: %v, want applied 0 and check_not_run 1", tl)
	}
}
