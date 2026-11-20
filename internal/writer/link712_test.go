package writer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// #712. Moving a relative symlink to another directory makes it read another
// file, but the receipt gave its destination the digest of the file it read at
// the old place, and the writer licensed that. A moved link's target is not read
// at all (the receipt, the drift scan and the ledger alike: Codex on #725), so
// its record carries no content and licenses nothing.

// moveLink builds a/payload.txt=A and b/payload.txt=B with a/link.txt ->
// payload.txt, runs prepare, renames the link to b/link.txt, and returns the
// result. It skips where symlinks cannot be made.
func moveLink(t *testing.T, prepare func(root string)) (string, apply.Result) {
	t.Helper()
	root := checkout(t)
	for name, body := range map[string]string{"a/payload.txt": "A\n", "b/payload.txt": "B\n"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.Dir(name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("payload.txt", filepath.Join(root, "a", "link.txt")); err != nil {
		t.Skipf("no symlinks here: %v", err)
	}
	if prepare != nil {
		prepare(root)
	}
	// The ledger as Land hands it to apply, so the writer's other-spelling
	// sweep runs.
	ledger, err := seen.Snapshot(root)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Apply(root, []apply.Input{{Path: "a/link.txt", Op: "rename", Body: []string{"b/link.txt"}, SrcLine: 1, Lines: -1, Index: 0}}, apply.Options{Force: true, Seen: ledger})
	if err != nil || !res.Applied {
		t.Fatalf("the rename did not land: %v %+v", err, res)
	}
	return root, res
}

// heldFor reports whether the ledger holds a record for p under either
// spelling (Windows keys may use the separator).
func heldFor(t *testing.T, root, p string) bool {
	t.Helper()
	l, err := seen.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	_, a := l[p]
	_, b := l[filepath.FromSlash(p)]
	return a || b
}

func TestAMovedSymlinkCarriesNoContentAndLicensesNothing(t *testing.T) {
	root, res := moveLink(t, nil)
	var found bool
	for _, f := range res.Files {
		if filepath.ToSlash(f.Path) == "b/link.txt" && f.Written && !f.Removed {
			found = true
			if f.SHAAfter != "" || f.LinesTo != 0 || !f.NotProduced {
				t.Errorf("the moved link's record carries content: %+v", f)
			}
		}
	}
	if !found {
		t.Fatalf("no record for b/link.txt: %+v", res.Files)
	}
	if heldFor(t, root, "b/link.txt") {
		t.Error("the ledger licenses b/link.txt, whose bytes nobody read")
	}
}

// A record an earlier read left at the destination's path is dropped: it could
// otherwise license an edit through the link (Codex on #725).
func TestAMovedSymlinkDropsAStaleRecordAtItsDestination(t *testing.T) {
	root, _ := moveLink(t, func(root string) {
		if err := seen.Record(root, map[string]seen.Observation{filepath.FromSlash("b/link.txt"): {SHA: seen.SHA([]byte("B\n"))}}); err != nil {
			t.Fatal(err)
		}
	})
	if heldFor(t, root, "b/link.txt") {
		t.Error("a stale record at the moved link's destination survived")
	}
}

// A stale record under another spelling of the same directory entry (c is a
// link to b, so c/link.txt names b/link.txt) is dropped too (Codex on #725).
func TestAMovedSymlinkDropsAStaleRecordUnderAnotherSpelling(t *testing.T) {
	root, _ := moveLink(t, func(root string) {
		if err := os.Symlink("b", filepath.Join(root, "c")); err != nil {
			t.Fatal(err)
		}
		if err := seen.Record(root, map[string]seen.Observation{filepath.FromSlash("c/link.txt"): {SHA: seen.SHA([]byte("B\n"))}}); err != nil {
			t.Fatal(err)
		}
	})
	if heldFor(t, root, "c/link.txt") {
		t.Error("a stale record of the moved link under another spelling survived")
	}
}

// A record of the file the moved link now points at is the caller's own read
// of unchanged bytes, and stays (Codex on #725: the other-spelling sweep
// followed the link and dropped it).
func TestAMovedSymlinkKeepsTheRecordOfTheFileItNowPointsAt(t *testing.T) {
	root, _ := moveLink(t, func(root string) {
		if err := seen.Record(root, map[string]seen.Observation{"b/payload.txt": {SHA: seen.SHA([]byte("B\n"))}}); err != nil {
			t.Fatal(err)
		}
	})
	if !heldFor(t, root, "b/payload.txt") {
		t.Error("the record of b/payload.txt, whose bytes never changed, was dropped")
	}
}

// A moved link that leads outside the root, or dangles, is not read either, and
// leaves nothing in the ledger file.
func TestAMovedSymlinkThatLeadsOutsideOrNowhereIsNotRead(t *testing.T) {
	outer := t.TempDir()
	root := filepath.Join(outer, "x", "R") // from root, ../../in.txt is outer/in.txt
	if err := os.MkdirAll(filepath.Join(root, "a", "b"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	for p, body := range map[string]string{filepath.Join(root, "in.txt"): "IN\n", filepath.Join(outer, "in.txt"): "SECRET\nline2\n"} {
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(filepath.Join("..", "..", "in.txt"), filepath.Join(root, "a", "b", "link.txt")); err != nil {
		t.Skipf("no symlinks here: %v", err)
	}
	// dangle.txt reads a/b/local.txt where it is, and nothing once it is moved.
	if err := os.WriteFile(filepath.Join(root, "a", "b", "local.txt"), []byte("L\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("local.txt", filepath.Join(root, "a", "b", "dangle.txt")); err != nil {
		t.Fatal(err)
	}
	res, err := Apply(root, []apply.Input{
		{Path: "a/b/link.txt", Op: "rename", Body: []string{"link.txt"}, SrcLine: 1, Lines: -1, Index: 0},
		{Path: "a/b/dangle.txt", Op: "rename", Body: []string{"dangle.txt"}, SrcLine: 3, Lines: -1, Index: 1},
	}, apply.Options{Force: true})
	if err != nil || !res.Applied {
		t.Fatalf("the renames did not land: %v %+v", err, res)
	}
	secret := seen.SHA([]byte("SECRET\nline2\n"))
	for _, f := range res.Files {
		if f.SHAAfter == secret || ((f.Path == "link.txt" || f.Path == "dangle.txt") && (f.SHAAfter != "" || f.LinesTo != 0)) {
			t.Errorf("a moved link's record carries content: %+v", f)
		}
	}
	// The drift scan may watch a moved link inside the root (ADR-112, the Codex
	// review of #309), but never reads one that leads outside it.
	if s := Before(root, res); s["link.txt"] != "" || s[filepath.FromSlash("link.txt")] != "" {
		t.Errorf("the drift snapshot hashed a target outside the root: %v", s)
	}
	p, err := seen.ReadPath(root)
	if err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(p); err == nil && (strings.Contains(string(b), "link.txt") || strings.Contains(string(b), "dangle.txt")) {
		t.Errorf("the ledger file names a moved link: %q", b)
	}
}
