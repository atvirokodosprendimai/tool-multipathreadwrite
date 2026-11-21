package apply

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// ADR-209 with ADR-210. The order of one write that makes asides and syncs: every
// staged temp is synced, then the asides are linked, then the renames, then the
// asides are removed, and only then are the changed directories synced, so the
// removal is made stable with the renames and no crash can leave an aside the synced
// state does not know about. The aside lives in its target's directory, which the
// write's recorded changed directories already hold.
func TestAsidesAndSyncsHaveOneOrder(t *testing.T) {
	u := newU209(t)
	var events []string
	rt, rs, rl, rr, rm := SyncTempFn, SyncDirFn, asideLinkFn, commitRenameFn, removeFn
	t.Cleanup(func() { SyncTempFn, SyncDirFn, asideLinkFn, commitRenameFn, removeFn = rt, rs, rl, rr, rm })
	SyncTempFn = func(fh *os.File) error { events = append(events, "temp"); return rt(fh) }
	SyncDirFn = func(fh *os.File) error { events = append(events, "dir"); return rs(fh) }
	asideLinkFn = func(tr *tree, from, to string) error { events = append(events, "link"); return rl(tr, from, to) }
	commitRenameFn = func(tr *tree, from, to string) error { events = append(events, "rename"); return rr(tr, from, to) }
	removeFn = func(tr *tree, p string) error {
		if strings.HasPrefix(filepath.Base(p), ".mrw-aside-") {
			events = append(events, "remove")
		}
		return rm(tr, p)
	}
	res, err := Apply(u.root, u.edit("a.txt", "b.txt"), Options{})
	if err != nil || !res.Applied {
		t.Fatalf("a plain write: %v", err)
	}
	lastIdx := func(e string) int {
		for i := len(events) - 1; i >= 0; i-- {
			if events[i] == e {
				return i
			}
		}
		return -1
	}
	for _, e := range []string{"temp", "link", "rename", "remove", "dir"} {
		if slices.Index(events, e) < 0 {
			t.Fatalf("no %s event: %v", e, events)
		}
	}
	switch {
	case lastIdx("temp") > slices.Index(events, "link"):
		t.Errorf("a temp was synced after an aside was made: %v", events)
	case lastIdx("link") > slices.Index(events, "rename"):
		t.Errorf("an aside was made after the first rename: %v", events)
	case lastIdx("rename") > slices.Index(events, "remove"):
		t.Errorf("an aside was removed before the last rename: %v", events)
	case lastIdx("remove") > slices.Index(events, "dir"):
		t.Errorf("a directory was synced before the asides were removed: %v", events)
	}
	u.noMrwEntries()
}

// A SyncError after landing is a landed write: the asides are already removed, so none
// is left behind and none needs naming.
func TestASyncErrorAfterLandingLeavesNoAside(t *testing.T) {
	u := newU209(t)
	rs := SyncDirFn
	t.Cleanup(func() { SyncDirFn = rs })
	SyncDirFn = func(*os.File) error { return errors.New("injected directory sync failure") }
	res, err := Apply(u.root, u.edit("a.txt", "b.txt"), Options{})
	var se *SyncError
	if !errors.As(err, &se) || !res.Applied {
		t.Fatalf("a failed directory sync after landing: applied %v, err %v", res.Applied, err)
	}
	u.noMrwEntries()
	if len(res.LeftBehind) != 0 {
		t.Errorf("left_behind = %v, want none: the asides went before the sync", res.LeftBehind)
	}
	if got := read(t, u.root, "a.txt"); got != "new a.txt\n" {
		t.Errorf("a.txt = %q, want the landed edit", got)
	}
}

// A commit that failed and was put back promises nothing of durability: no directory
// is synced, since nothing was written, and NOTHING WRITTEN is true of the page cache
// and of the disk alike.
func TestAFailedCommitThatWasPutBackSyncsNoDirectory(t *testing.T) {
	u := newU209(t)
	u.refuse("b.txt")
	calls := 0
	rs := SyncDirFn
	t.Cleanup(func() { SyncDirFn = rs })
	SyncDirFn = func(fh *os.File) error { calls++; return rs(fh) }
	res, err := Apply(u.root, u.edit("a.txt", "b.txt"), Options{})
	if err == nil || res.Applied {
		t.Fatalf("a refused rename: %v", err)
	}
	if calls != 0 {
		t.Errorf("%d directory sync(s) after a commit that was put back", calls)
	}
	u.restored("a.txt")
}
