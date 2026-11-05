package writer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// Issue #597. The ledger is keyed by the spelled path, and one file can carry
// more than one spelling: a case variant on a case-insensitive filesystem, or
// an in-root symlink anywhere. A write under one spelling recorded the new
// content under that spelling only, so the entry kept under the other still
// held the old sha, and a write through it was refused as "changed since mrw
// last saw it" — a false refusal on the file mrw itself had just written.

const lines597 = "1\n2\n3\n4\n5\n6\n"

// caseFolds reports whether dir's filesystem treats two spellings of one name
// as one file. Asked, not assumed from the OS.
func caseFolds(t *testing.T, dir string) bool {
	t.Helper()
	p := filepath.Join(dir, "CaseProbe.tmp")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(p)
	_, err := os.Stat(filepath.Join(dir, "caseprobe.tmp"))
	return err == nil
}

// land597 replaces one line the way Land does: the ledger is loaded before the
// landing and handed to it.
func land597(t *testing.T, root, path string, line int, text string) apply.Result {
	t.Helper()
	l, err := seen.Snapshot(root)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Apply(root, []apply.Input{
		{Path: path, Start: line, End: line, Op: "replace", Body: []string{text}, Lines: -1, SrcLine: 1, Index: 0},
	}, apply.Options{Seen: l})
	if err != nil {
		t.Fatalf("write %s:%d: %v", path, line, err)
	}
	return res
}

func readSpans597(t *testing.T, root, path string, from, to int) {
	t.Helper()
	if err := seen.Record(root, map[string]seen.Observation{path: {SHA: seen.SHA([]byte(lines597)), Spans: [][2]int{{from, to}}}}); err != nil {
		t.Fatal(err)
	}
}

func wantFile597(t *testing.T, root, name, want string) {
	t.Helper()
	got, err := os.ReadFile(filepath.Join(root, name))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("%s holds %q, want %q", name, got, want)
	}
}

// One file, two spellings: read as A.TXT, written as a.txt, then written as
// A.TXT again. The second write must land, and the ledger must not keep an
// entry for the file at a sha it no longer has.
func TestAWriteUnderOneSpellingDropsTheEntryKeptUnderAnother(t *testing.T) {
	root := checkout(t)
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte(lines597), 0o644); err != nil {
		t.Fatal(err)
	}
	if !caseFolds(t, root) {
		t.Skip("needs a case-insensitive filesystem (NTFS; APFS by default); the symlink test covers the same path elsewhere")
	}
	readSpans597(t, root, "A.TXT", 5, 6)

	if res := land597(t, root, "a.txt", 5, "five"); !res.Applied {
		t.Fatalf("the first write, as a.txt, did not land: %+v", res.Hunks)
	}
	if res := land597(t, root, "A.TXT", 6, "six"); !res.Applied {
		t.Fatalf("a write as A.TXT after mrw wrote the file as a.txt was refused: %+v", res.Hunks)
	}
	wantFile597(t, root, "a.txt", "1\n2\n3\n4\nfive\nsix\n")

	l, err := seen.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(l) != 1 {
		t.Fatalf("one file is held under %d ledger entries: %+v", len(l), l)
	}
}

// The same defect through a symlink, which a case-sensitive filesystem can
// reach: read as link.txt, written as a.txt, then written as link.txt.
func TestAWriteThroughATargetDropsTheEntryKeptUnderItsLink(t *testing.T) {
	root := checkout(t)
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte(lines597), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("a.txt", filepath.Join(root, "link.txt")); err != nil {
		t.Skipf("symlinks are refused here: %v", err)
	}
	readSpans597(t, root, "link.txt", 5, 6)

	if res := land597(t, root, "a.txt", 5, "five"); !res.Applied {
		t.Fatalf("the first write, as a.txt, did not land: %+v", res.Hunks)
	}
	if res := land597(t, root, "link.txt", 6, "six"); !res.Applied {
		t.Fatalf("a write as link.txt after mrw wrote its target was refused: %+v", res.Hunks)
	}
	wantFile597(t, root, "a.txt", "1\n2\n3\n4\nfive\nsix\n")
}

// The valid sibling: where a.txt and A.TXT are two files, a write to one must
// leave the other's entry alone — folding case would drop it.
func TestAWriteKeepsTheEntryOfADifferentFileWhoseNameDiffersOnlyInCase(t *testing.T) {
	root := checkout(t)
	for _, name := range []string{"a.txt", "A.TXT"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(lines597), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if caseFolds(t, root) {
		t.Skip("needs a case-sensitive filesystem, where a.txt and A.TXT are two files")
	}
	readSpans597(t, root, "a.txt", 5, 6)
	readSpans597(t, root, "A.TXT", 5, 6)

	if res := land597(t, root, "a.txt", 5, "five"); !res.Applied {
		t.Fatalf("the write to a.txt did not land: %+v", res.Hunks)
	}
	l, err := seen.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if o, ok := l["A.TXT"]; !ok || o.SHA != seen.SHA([]byte(lines597)) {
		t.Fatalf("a write to a.txt changed the entry of the distinct file A.TXT: %+v", l)
	}
}
