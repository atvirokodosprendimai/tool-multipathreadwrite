package state

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The three defects the 2026-09-07 review found in the prune, each written red
// before the fix. A fourth finding — that the fixtures leave `self` alive, so
// the self guard is uncovered — did NOT hold: deleting `e.Dir == self` from
// Prune is killed by TestOnlyAnEntryWhoseCheckoutIsGoneIsPruned and
// TestADryRunPruneReportsTheSameAndRemovesNothing, because plant removes
// selfGo. Measured, not assumed; there is no test here for it because the
// guard already has two.

// TestABaseThatIsASymlinkIsRefusedRatherThanFollowed is the HIGH. Entries and
// Prune reach the base by PATH, so `<state>/mrw` being a symlink puts a
// directory nobody gave mrw inside the prune's reach: os.ReadDir follows the
// link to enumerate, and os.RemoveAll follows it again to delete. The existing
// TestThePruneStaysInsideTheStateBase covers a symlinked ENTRY under a real
// base, which is a different object — the base itself is never checked.
func TestABaseThatIsASymlinkIsRefusedRatherThanFollowed(t *testing.T) {
	base := xdg(t)

	// A directory that is not mrw's, holding what looks like a prunable entry:
	// a `root` marker naming a checkout that does not exist.
	victim := t.TempDir()
	entry := filepath.Join(victim, "4444444444444444")
	if err := os.MkdirAll(entry, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(entry, "root"), []byte("/no/such/checkout\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(victim, "precious"), []byte("not mrw's\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(victim, filepath.Join(base, "mrw")); err != nil {
		t.Skipf("this filesystem does not do symlinks: %v", err)
	}

	entries, err := Entries()
	if err == nil {
		// Not an error is survivable only if the walk found nothing to remove.
		// What must never happen is a removal, which is asserted below.
		t.Logf("Entries returned %d entries through a symlinked base", len(entries))
	}
	removed, err := Prune(t.TempDir(), entries, false)
	if err == nil {
		t.Fatalf("Prune accepted a symlinked base and reported %d removals; "+
			"a base mrw did not create is one it must refuse by name", len(removed))
	}
	// ⚠ THE MESSAGE IS THE ASSERTION, and this is the whole reason the explicit
	// check exists. Containment is NOT what the check buys: os.Root refuses a
	// symlinked child on its own with "openat mrw: path escapes from parent",
	// so deleting the check still refuses the removal — measured, and the
	// mutant survived until this clause was written. What the check buys is a
	// refusal that names the base and says what is wrong with it, which
	// ADR-015 is about. A test that only asserted "some error" would have
	// scored a guard that could not fail.
	if !strings.Contains(err.Error(), "symlink") || !strings.Contains(err.Error(), filepath.Join(base, "mrw")) {
		t.Errorf("the refusal is %q; it must name the base %q and say it is a symlink, "+
			"because os.Root's own message says only that a path escaped",
			err, filepath.Join(base, "mrw"))
	}

	if !exists(t, entry) {
		t.Errorf("Prune followed a symlinked base and removed %q, which is not mrw's to touch", entry)
	}
	if !exists(t, filepath.Join(victim, "precious")) {
		t.Errorf("Prune removed %q through a symlinked base", filepath.Join(victim, "precious"))
	}
}

// TestAMarkerIsReadBackExactlyAsDirWroteIt covers the MEDIUM in describe.
// Dir writes the path plus exactly one "\n"; describe strips with TrimSpace,
// which is wider. A checkout whose directory name ends in a space therefore
// reads back as a DIFFERENT path — one that usually does not exist, so a live
// checkout is classified dead and removed.
func TestAMarkerIsReadBackExactlyAsDirWroteIt(t *testing.T) {
	xdg(t)

	// A real, live checkout whose name ends in a space. Legal on every
	// filesystem this runs on; only the reader is wrong about it.
	parent := t.TempDir()
	root := filepath.Join(parent, "checkout ")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Skipf("this filesystem does not allow a trailing space in a name: %v", err)
	}
	want, err := absReal(root)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(want, " ") {
		t.Skipf("this filesystem normalised the trailing space away: %q", want)
	}
	if _, err := Dir(root); err != nil {
		t.Fatal(err)
	}

	entries, err := Entries()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("planted one entry, Entries found %d", len(entries))
	}
	e := entries[0]
	if e.Root != want {
		t.Errorf("the marker reads back as %q, want %q — Dir wrote exactly one newline, "+
			"so exactly one is what may be stripped", e.Root, want)
	}
	if !e.Identified {
		t.Errorf("a marker mrw itself wrote reads as unidentified")
	}

	removed, err := Prune(t.TempDir(), entries, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 0 {
		t.Errorf("Prune removed %d entries; the checkout %q is still there", len(removed), root)
	}
	if !exists(t, e.Dir) {
		t.Errorf("Prune removed %q, the state of a checkout that exists", e.Dir)
	}
}

// TestARootThatCannotBeStattedIsKept covers the other half of the same MEDIUM.
// describe treats EVERY stat error as "gone", but only ErrNotExist means gone.
// A permission error, a timed-out network mount, an I/O error — each is
// indeterminate, and removing state on an indeterminate answer is the one
// mistake here that cannot be undone by looking again.
func TestARootThatCannotBeStattedIsKept(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permissions do not deny traversal the same way on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root: mode 0 does not deny traversal")
	}
	xdg(t)

	parent := t.TempDir()
	root := filepath.Join(parent, "checkout")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := Dir(root); err != nil {
		t.Fatal(err)
	}

	// Deny traversal of the parent, so stat of the root fails with EACCES
	// rather than ENOENT. The checkout is still there.
	if err := os.Chmod(parent, 0o000); err != nil {
		t.Skipf("cannot deny traversal here: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o700) })
	if _, err := os.Stat(root); err == nil {
		t.Skip("this filesystem ignores the mode; stat still succeeds")
	} else if os.IsNotExist(err) {
		t.Skipf("stat reported not-exist rather than a permission error: %v", err)
	}

	entries, err := Entries()
	if err != nil {
		t.Fatal(err)
	}
	removed, err := Prune(t.TempDir(), entries, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 0 {
		t.Errorf("Prune removed %d entries on a stat error that is not ErrNotExist; "+
			"indeterminate must mean keep", len(removed))
	}
}

// TestAnEntryThatVanishedBetweenTheWalkAndTheRemovalIsNotReported is what
// makes Prune's "only what this walk saw" clause a guard rather than a
// comment. The entries a caller hands back describe a PAST walk; between that
// walk and the removal an entry can go — another mrw, an operator, a tmp
// reaper. os.Root.RemoveAll succeeds silently on a name that is not there, so
// without the check the run reports a removal it did not perform, and ADR-008
// says a delete says what it removed.
func TestAnEntryThatVanishedBetweenTheWalkAndTheRemovalIsNotReported(t *testing.T) {
	p := plant(t)

	entries, err := Entries()
	if err != nil {
		t.Fatal(err)
	}

	// The dead entry goes away after the walk and before the prune.
	if err := os.RemoveAll(p.deadDir); err != nil {
		t.Fatal(err)
	}

	removed, err := Prune(p.selfGo, entries, false)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	for _, e := range removed {
		if e.Dir == p.deadDir {
			t.Errorf("Prune reported removing %q, which was already gone when it looked; "+
				"RemoveAll succeeding on a missing name is not a removal", e.Dir)
		}
	}
	if len(removed) != 0 {
		t.Errorf("Prune reported %d removals, want 0 — the only prunable entry had vanished", len(removed))
	}
}

// TestSelfSurvivesWhenItsCheckoutIsGoneUnderASymlinkedPath covers the third
// MEDIUM. Dir keys a checkout by absReal, which resolves symlinks WHILE the
// path exists and falls back to filepath.Abs once it does not. So for a root
// reached through a symlink, the key computed after deletion is not the key
// the directory was created under, selfDir stops matching, and the prune
// removes the state of the run in progress — precisely the case Prune's doc
// comment promises to protect.
//
// The existing fixture cannot see this: plant resolves every root through
// EvalSymlinks before handing it to Dir, so the two spellings coincide.
func TestSelfSurvivesWhenItsCheckoutIsGoneUnderASymlinkedPath(t *testing.T) {
	xdg(t)

	tmp := t.TempDir()
	real := filepath.Join(tmp, "real")
	if err := os.MkdirAll(filepath.Join(real, "checkout"), 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(tmp, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("this filesystem does not do symlinks: %v", err)
	}
	// The caller names the checkout through the link, which is what a shell
	// hands a program on any machine where a path component is a symlink —
	// /var on macOS being the everyday case.
	root := filepath.Join(link, "checkout")

	dir, err := Dir(root)
	if err != nil {
		t.Fatal(err)
	}
	// The checkout goes away underneath the run.
	if err := os.RemoveAll(filepath.Join(real, "checkout")); err != nil {
		t.Fatal(err)
	}

	entries, err := Entries()
	if err != nil {
		t.Fatal(err)
	}
	removed, err := Prune(root, entries, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range removed {
		if e.Dir == dir {
			t.Fatalf("Prune removed the state of the run in progress (%q); "+
				"the checkout was named through a symlink, so absReal spells it "+
				"one way while it exists and another once it does not", dir)
		}
	}
	if !exists(t, dir) {
		t.Errorf("the caller's own entry %q is gone", dir)
	}
}
