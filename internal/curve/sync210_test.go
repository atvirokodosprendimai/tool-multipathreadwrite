package curve

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
)

// ADR-210 review. A plan whose directory sync failed AFTER it landed did apply:
// the scorer scores what it changed and does not call it a refusal.
func TestAPlanWhoseSyncFailedAfterLandingIsScoredAsApplied(t *testing.T) {
	rs, rf := apply.SyncsDirs, apply.SyncDirFn
	t.Cleanup(func() { apply.SyncsDirs, apply.SyncDirFn = rs, rf })
	apply.SyncsDirs = true
	apply.SyncDirFn = func(*os.File) error { return errors.New("injected sync failure") }
	_, m, a := cell(t, Params{ServedBytes: 4000, Position: Middle, Distractors: 4, Seed: 1})
	r := ok(m)
	r.Plan = replaceAt(m, a.Line)
	s, err := ScoreTrial(filepath.Dir(m.Tree), r)
	if err != nil {
		t.Fatal(err)
	}
	if s.Outcome != Hit {
		t.Errorf("a landed plan whose sync failed scored %s (%s), want a hit", s.Outcome, s.Reason)
	}
}

// ADR-210 review. A staged temp that could not be made stable stopped the plan
// before anything landed: that is the storage's failure, so the trial is not
// scored at all (an error), and is not counted as a refusal of the plan.
func TestAStagingSyncFailureIsTheHarnessesNotARefusal(t *testing.T) {
	rt := apply.SyncTempFn
	t.Cleanup(func() { apply.SyncTempFn = rt })
	apply.SyncTempFn = func(*os.File) error { return errors.New("injected sync failure") }
	_, m, a := cell(t, Params{ServedBytes: 4000, Position: Middle, Distractors: 4, Seed: 1})
	r := ok(m)
	r.Plan = replaceAt(m, a.Line)
	s, err := ScoreTrial(filepath.Dir(m.Tree), r)
	if !errors.Is(err, apply.ErrStagingSync) || s.Outcome == RefusedApply {
		t.Errorf("outcome %q, err %v; want an error naming the staging sync and no scored outcome", s.Outcome, err)
	}
}
