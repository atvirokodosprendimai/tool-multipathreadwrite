package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/authoring"
)

// ADR-083. A plan the CLI refused after it parsed and before anything landed
// went uncounted: the write action returned before its ADR-009 tally, while
// mrw_write counted the same refusal as refused_apply. Each member of the
// class is one refusal; a refusal before the plan parsed is none; a clean
// write beside them is applied.
func TestAPlanRefusedAfterItParsedIsOneRefusal(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(root, n), []byte(n+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := readIn(t, root, "a.txt", "b.txt"); err != nil {
		t.Fatal(err)
	}
	if out, code := runIn(t, root, "iter", "add", "a.txt", "b.txt"); code != 0 {
		t.Fatalf("iter add: exit %d\n%s", code, out)
	}
	want := func(step string, refused, applied, plans int) {
		t.Helper()
		tally, err := authoring.Load(root)
		if err != nil {
			t.Fatal(err)
		}
		if tally["refused_apply"] != refused || tally["applied"] != applied || tally.Plans() != plans {
			t.Errorf("%s: tally %v, want refused_apply %d, applied %d, plans %d", step, tally, refused, applied, plans)
		}
	}

	dir := planFile(t, "@@ d 1 replace\nx\n")
	if out, code := writeIn(t, root, "--no-check", dir); code != 2 {
		t.Fatalf("a plan naming a directory: exit %d, want 2\n%s", code, out)
	}
	want("a filesystem refusal", 1, 0, 1)

	if out, code := writeIn(t, root, "--no-check", "--dry-run", dir); code != 2 {
		t.Fatalf("the same plan under --dry-run: exit %d, want 2\n%s", code, out)
	}
	want("a refused dry run", 2, 0, 2)

	if out, code := writeIn(t, root, "--no-check", planFile(t, "@@ @1-2 1 replace\nx\n")); code != 2 {
		t.Fatalf("a pointer naming two entries: exit %d, want 2\n%s", code, out)
	}
	want("a pointer refused after the plan parsed", 3, 0, 3)

	if out, code := writeIn(t, root, "--no-check", filepath.Join(t.TempDir(), "missing.mrw")); code != 2 {
		t.Fatalf("a missing plan file: exit %d, want 2\n%s", code, out)
	}
	want("a plan file that could not be read", 3, 0, 3)

	if out, code := writeIn(t, root, "--no-check", planFile(t, "@@ a.txt 1 replace\nA\n")); code != 0 {
		t.Fatalf("a clean write: exit %d, want 0\n%s", code, out)
	}
	want("a clean write beside them", 3, 1, 4)
}
