package apply

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// editRenamePlan edits line 2 of a.go and renames it to m/b.go: one plan,
// the shape Codex apply_patch sends as Update File + Move to + a hunk.
func editRenamePlan() []Input {
	return []Input{
		{Path: "a.go", Start: 2, End: 2, Op: "replace", Body: []string{"B"}, Lines: -1, SrcLine: 1, Index: 0},
		{Path: "a.go", Op: "rename", Body: []string{"m/b.go"}, Lines: -1, SrcLine: 3, Index: 1},
	}
}

// recordsFor counts res.Files records per path, and returns the last one.
func recordsFor(res Result, path string) (int, FileResult) {
	n, last := 0, FileResult{}
	for _, f := range res.Files {
		if f.Path == path {
			n++
			last = f
		}
	}
	return n, last
}

func exists114(t *testing.T, root, name string) bool {
	t.Helper()
	_, err := os.Lstat(filepath.Join(root, name))
	return err == nil
}

// ADR-114 T1. The engine refused any rename beside another hunk on the same
// path, so a move-and-edit took two plans, and the pair was not atomic. A
// plan may now carry line edits and one rename of a file: the edit lands at
// the destination, the receipt names each path once, and a failure that
// undoes the rename leaves the source named as the edit that landed.
func TestAFileIsEditedAndRenamedInOnePlan(t *testing.T) {
	const edited = "a\nB\nc\nd\ne\n"
	whole := func(t *testing.T, root string) Options {
		return Options{Seen: map[string]Seen{"a.go": {SHA: shaOfFile(t, root, "a.go")}}}
	}

	t.Run("the edit lands at the destination, each path named once", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, "a.go", abcde)
		res, err := Apply(root, editRenamePlan(), whole(t, root))
		if err != nil || !res.Applied {
			t.Fatalf("err %v applied %v %+v", err, res.Applied, res.Hunks)
		}
		if got := read(t, root, "m/b.go"); got != edited {
			t.Errorf("m/b.go holds %q, want %q", got, edited)
		}
		if exists114(t, root, "a.go") {
			t.Error("a.go is still there")
		}
		n, src := recordsFor(res, "a.go")
		if n != 1 || !src.Written || !src.Removed || src.RenamedTo != "m/b.go" || src.SHAAfter != "" {
			t.Errorf("a.go: %d record(s), last %+v; want one, removed, renamed to m/b.go", n, src)
		}
		n, dst := recordsFor(res, "m/b.go")
		if n != 1 || !dst.Written || dst.SHAAfter != shaOfFile(t, root, "m/b.go") || dst.LinesTo != 5 {
			t.Errorf("m/b.go: %d record(s), last %+v; want one, written, at the edited sha, 5 lines", n, dst)
		}
	})

	t.Run("a dry run lists the source once and writes nothing", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, "a.go", abcde)
		opt := whole(t, root)
		opt.DryRun = true
		res, err := Apply(root, editRenamePlan(), opt)
		if err != nil || res.Failed != 0 {
			t.Fatalf("err %v %+v", err, res.Hunks)
		}
		if n, src := recordsFor(res, "a.go"); n != 1 || src.RenamedTo != "m/b.go" {
			t.Errorf("a.go: %d record(s), last %+v; want one naming m/b.go", n, src)
		}
		if read(t, root, "a.go") != abcde || exists114(t, root, "m/b.go") {
			t.Error("a dry run changed the tree")
		}
	})

	t.Run("an unlink beside an edit, and two renames, stay refused", func(t *testing.T) {
		for name, in := range map[string][]Input{
			"unlink": {
				{Path: "a.go", Start: 2, End: 2, Op: "replace", Body: []string{"B"}, Lines: -1, Index: 0},
				{Path: "a.go", Op: "unlink", Lines: -1, Index: 1},
			},
			"two renames": {
				{Path: "a.go", Op: "rename", Body: []string{"x.go"}, Lines: -1, Index: 0},
				{Path: "a.go", Op: "rename", Body: []string{"y.go"}, Lines: -1, Index: 1},
			},
		} {
			root := t.TempDir()
			write(t, root, "a.go", abcde)
			res, err := Apply(root, in, whole(t, root))
			if err != nil || res.Applied || res.Failed == 0 || read(t, root, "a.go") != abcde {
				t.Errorf("%s: err %v applied %v failed %d; want refused, a.go unchanged", name, err, res.Applied, res.Failed)
			}
		}
	})

	t.Run("a partial read refuses the rename", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, "a.go", abcde)
		res, err := Apply(root, editRenamePlan(), Options{Seen: map[string]Seen{"a.go": {SHA: shaOfFile(t, root, "a.go"), Spans: [][2]int{{2, 2}}}}})
		if err != nil || res.Applied || res.Failed == 0 || read(t, root, "a.go") != abcde || exists114(t, root, "m/b.go") {
			t.Errorf("err %v applied %v %+v; want the rename refused and nothing written", err, res.Applied, res.Hunks)
		}
	})

	t.Run("a rename that fails leaves the edit landed and named", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, "a.go", abcde)
		real := commitRenameFn
		t.Cleanup(func() { commitRenameFn = real })
		commitRenameFn = func(tr *tree, from, to string) error {
			if filepath.Base(from) == "a.go" && filepath.Base(to) == "b.go" {
				return errors.New("rename refused")
			}
			return real(tr, from, to)
		}
		res, err := Apply(root, editRenamePlan(), whole(t, root))
		if err == nil {
			t.Fatal("the failed rename returned no error")
		}
		if read(t, root, "a.go") != edited {
			t.Fatalf("a.go holds %q, want the edit that landed", read(t, root, "a.go"))
		}
		if n, src := recordsFor(res, "a.go"); n != 1 || !src.Written || src.Removed || src.SHAAfter != shaOfFile(t, root, "a.go") {
			t.Errorf("a.go: %d record(s), last %+v; want one, written with the edited sha, not removed", n, src)
		}
		if res.Hunks[0].Status != StatusOK || res.Hunks[1].Status != StatusFailed {
			t.Errorf("verdicts %s/%s, want the edit ok and the rename failed", res.Hunks[0].Status, res.Hunks[1].Status)
		}
	})

	t.Run("a later failure undoes the rename and restores the edited record", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, "a.go", abcde)
		write(t, root, "c.txt", "c\n")
		real := commitRenameFn
		t.Cleanup(func() { commitRenameFn = real })
		commitRenameFn = func(tr *tree, from, to string) error {
			if filepath.Base(from) == "c.txt" && filepath.Base(to) == "d.txt" {
				return errors.New("rename refused")
			}
			return real(tr, from, to)
		}
		in := append(editRenamePlan(), Input{Path: "c.txt", Op: "rename", Body: []string{"d.txt"}, Lines: -1, SrcLine: 5, Index: 2})
		opt := whole(t, root)
		opt.Seen["c.txt"] = Seen{SHA: shaOfFile(t, root, "c.txt")}
		res, err := Apply(root, in, opt)
		if err == nil {
			t.Fatal("the failed rename returned no error")
		}
		if read(t, root, "a.go") != edited || exists114(t, root, "m/b.go") {
			t.Fatalf("a.go %q, m/b.go there %v; want the edit back at a.go", read(t, root, "a.go"), exists114(t, root, "m/b.go"))
		}
		if n, src := recordsFor(res, "a.go"); n != 1 || !src.Written || src.Removed || src.SHAAfter != shaOfFile(t, root, "a.go") {
			t.Errorf("a.go: %d record(s), last %+v; want one, written with the edited sha", n, src)
		}
		if n, _ := recordsFor(res, "m/b.go"); n != 0 {
			t.Errorf("m/b.go still has %d record(s) after the undo", n)
		}
		if res.Hunks[0].Status != StatusOK || res.Hunks[1].Status != StatusSkipped || res.Hunks[2].Status != StatusFailed {
			t.Errorf("verdicts %s/%s/%s, want ok/skipped/failed", res.Hunks[0].Status, res.Hunks[1].Status, res.Hunks[2].Status)
		}
	})
}
