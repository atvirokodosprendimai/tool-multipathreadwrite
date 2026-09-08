package state

import (
	"os"
	"path/filepath"
	"testing"
)

// planted is the state base built by plant, with the entry directory of each
// class the prune has to tell apart.
type planted struct {
	base string // $XDG_STATE_HOME
	mrw  string // <base>/mrw

	liveDir string // its root exists
	liveGo  string
	deadDir string // its root was removed
	deadGo  string
	selfDir string // the root Prune is called with, and it has been removed too
	selfGo  string

	noMarker    string // no `root` file at all
	emptyMarker string // a `root` file holding nothing
	relMarker   string // a `root` file holding a relative path
}

// plant builds one state base holding every class of entry at once.
//
// ⚠ THE POINT IS THAT THEY SHARE ONE BASE. A fixture with only a dead entry is
// green against a Prune that removes everything, which is this record's
// pre-registered High-likelihood failure — the whole test then proves that
// os.RemoveAll works. Every assertion below names both a thing that must go and
// a thing that must stay.
func plant(t *testing.T) planted {
	t.Helper()
	p := planted{base: xdg(t)}
	p.mrw = filepath.Join(p.base, "mrw")

	// Three real entries, created the way mrw creates them: by asking Dir,
	// which is what writes the `root` marker this whole design reads.
	// ⚠ RESOLVE THE ROOTS THE WAY Dir DOES. absReal runs EvalSymlinks, and on
	// macOS t.TempDir() hands back /var/folders/... whose real path is
	// /private/var/folders/... — so a marker never equals the string the test
	// planted unless the test resolves it too.
	p.liveGo = resolved(t, t.TempDir())
	p.deadGo = resolved(t, t.TempDir())
	p.selfGo = resolved(t, t.TempDir())
	for _, pair := range []struct {
		root string
		into *string
	}{{p.liveGo, &p.liveDir}, {p.deadGo, &p.deadDir}, {p.selfGo, &p.selfDir}} {
		dir, err := Dir(pair.root)
		if err != nil {
			t.Fatalf("Dir(%q): %v", pair.root, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "seen"), []byte("{}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		*pair.into = dir
	}
	// Two of the three checkouts are gone. selfGo is gone AND is the root the
	// caller is running in, which is the case an "it does not exist, remove it"
	// rule gets wrong.
	for _, gone := range []string{p.deadGo, p.selfGo} {
		if err := os.RemoveAll(gone); err != nil {
			t.Fatal(err)
		}
	}

	// Three entries of unknown provenance. Hand-built, because Dir cannot
	// produce them — which is the point: mrw did not write these, so mrw does
	// not know what they are.
	for _, spec := range []struct {
		name   string
		marker string
		write  bool
		into   *string
	}{
		{name: "0000000000000000", write: false, into: &p.noMarker},
		{name: "1111111111111111", marker: "  \n", write: true, into: &p.emptyMarker},
		{name: "2222222222222222", marker: "relative/not/absolute\n", write: true, into: &p.relMarker},
	} {
		dir := filepath.Join(p.mrw, spec.name)
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		if spec.write {
			if err := os.WriteFile(filepath.Join(dir, "root"), []byte(spec.marker), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		*spec.into = dir
	}
	return p
}

func exists(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Stat(path)
	return err == nil
}

// resolved is t.TempDir() as absReal would spell it.
func resolved(t *testing.T, dir string) string {
	t.Helper()
	real, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", dir, err)
	}
	return real
}

func TestOnlyAnEntryWhoseCheckoutIsGoneIsPruned(t *testing.T) {
	p := plant(t)

	removed, err := PruneAll(t, p.selfGo, false)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}

	if len(removed) != 1 {
		var got []string
		for _, e := range removed {
			got = append(got, e.Dir+" ← "+e.Root)
		}
		t.Fatalf("Prune removed %d entries, want exactly 1 (the dead one): %v", len(removed), got)
	}
	if removed[0].Dir != p.deadDir {
		t.Errorf("Prune removed %q, want the dead entry %q", removed[0].Dir, p.deadDir)
	}
	if removed[0].Root != p.deadGo {
		t.Errorf("the removed entry names root %q, want %q — ADR-008: a delete says what it removed",
			removed[0].Root, p.deadGo)
	}
	if removed[0].Err != nil {
		t.Errorf("removal reported an error: %v", removed[0].Err)
	}

	if exists(t, p.deadDir) {
		t.Errorf("the dead entry %q is still on disk; Prune reported it but did not remove it", p.deadDir)
	}
	for what, dir := range map[string]string{
		"the live entry":                 p.liveDir,
		"the caller's own entry":         p.selfDir,
		"the entry with no marker":       p.noMarker,
		"the entry with an empty marker": p.emptyMarker,
		"the entry with a relative path": p.relMarker,
	} {
		if !exists(t, dir) {
			t.Errorf("%s (%q) was removed; only an entry whose marker names a missing path may go", what, dir)
		}
	}
}

func TestADryRunPruneReportsTheSameAndRemovesNothing(t *testing.T) {
	p := plant(t)

	dry, err := PruneAll(t, p.selfGo, true)
	if err != nil {
		t.Fatalf("Prune dry: %v", err)
	}
	if len(dry) != 1 || dry[0].Dir != p.deadDir {
		t.Fatalf("a dry run reported %d entries, want the one dead entry %q", len(dry), p.deadDir)
	}
	if !exists(t, p.deadDir) {
		t.Fatalf("a dry run removed %q", p.deadDir)
	}

	// The contract that makes --dry-run a preview rather than a different
	// question: the real run reports what the dry run promised.
	real, err := PruneAll(t, p.selfGo, false)
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if len(real) != len(dry) || real[0].Dir != dry[0].Dir {
		t.Errorf("the real run removed %v, the dry run promised %v", real, dry)
	}
	if exists(t, p.deadDir) {
		t.Errorf("the real run did not remove %q", p.deadDir)
	}
}

func TestAnUnidentifiableEntryIsKeptAndReported(t *testing.T) {
	p := plant(t)

	entries, err := Entries()
	if err != nil {
		t.Fatalf("Entries: %v", err)
	}
	byDir := map[string]Entry{}
	for _, e := range entries {
		byDir[e.Dir] = e
	}

	// Kept is not enough. An entry nothing reports is indistinguishable from
	// one nothing looked at, and the command layer can only print what it is
	// handed.
	for what, dir := range map[string]string{
		"no marker":       p.noMarker,
		"an empty marker": p.emptyMarker,
		"a relative path": p.relMarker,
	} {
		e, ok := byDir[dir]
		if !ok {
			t.Errorf("the entry with %s (%q) is not in Entries() at all", what, dir)
			continue
		}
		if e.Identified {
			t.Errorf("the entry with %s claims to be identified, naming root %q", what, e.Root)
		}
		if e.Dead {
			t.Errorf("the entry with %s claims its checkout is gone", what)
		}
	}

	if e := byDir[p.liveDir]; !e.Identified || e.Dead || e.Root != p.liveGo {
		t.Errorf("the live entry reads identified=%v dead=%v root=%q, want true false %q",
			e.Identified, e.Dead, e.Root, p.liveGo)
	}
	if e := byDir[p.deadDir]; !e.Identified || !e.Dead || e.Root != p.deadGo {
		t.Errorf("the dead entry reads identified=%v dead=%v root=%q, want true true %q",
			e.Identified, e.Dead, e.Root, p.deadGo)
	}
	if e := byDir[p.liveDir]; e.Bytes <= 0 {
		t.Errorf("the live entry reports %d bytes; it holds a marker and a ledger", e.Bytes)
	}
}

func TestThePruneStaysInsideTheStateBase(t *testing.T) {
	p := plant(t)

	// A directory OUTSIDE the base, reachable from inside it by a symlink whose
	// own "root" marker names something that is gone. If the walk follows the
	// link, it removes a directory that is not mrw's to touch.
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "root"), []byte("/no/such/checkout\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "precious"), []byte("not mrw's\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(p.mrw, "3333333333333333")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("this filesystem does not do symlinks: %v", err)
	}
	// A plain FILE under the base, which is not an entry either.
	stray := filepath.Join(p.mrw, "notadirectory")
	if err := os.WriteFile(stray, []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := PruneAll(t, p.selfGo, false); err != nil {
		t.Fatalf("Prune: %v", err)
	}

	if !exists(t, outside) || !exists(t, filepath.Join(outside, "precious")) {
		t.Errorf("Prune followed a symlink out of %q and removed %q", p.mrw, outside)
	}
	if !exists(t, link) {
		t.Errorf("Prune removed the symlink %q; an entry it cannot identify is one it leaves alone", link)
	}
	if !exists(t, stray) {
		t.Errorf("Prune removed the plain file %q; only directories are entries", stray)
	}
}

// PruneAll is Entries + Prune, which is what every caller does: Prune selects
// from a walk it does not perform, so that the command layer can report what
// was KEPT from the same slice without walking a second time.
func PruneAll(t *testing.T, root string, dryRun bool) ([]Entry, error) {
	t.Helper()
	entries, err := Entries()
	if err != nil {
		return nil, err
	}
	return Prune(root, entries, dryRun)
}
