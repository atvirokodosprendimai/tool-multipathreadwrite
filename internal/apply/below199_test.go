package apply

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// snapshot records every entry under root, a link by its target and a file by
// its bytes, so a refused plan can be shown to have changed nothing.
func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			to, _ := os.Readlink(p)
			out[rel] = "link " + to
		case info.IsDir():
			out[rel] = "dir"
		case info.Mode().IsRegular():
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			out[rel] = "file " + string(b)
		default:
			out[rel] = info.Mode().String()
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// noStaging fails the test if a plan reaches staging: a refusal that comes from
// validation stages nothing.
func noStaging(t *testing.T) {
	t.Helper()
	real := stageFileFn
	t.Cleanup(func() { stageFileFn = real })
	stageFileFn = func(*tree, string, text) (staged, error) {
		t.Fatal("staging was entered by a plan validation should have refused")
		return staged{}, nil
	}
}

// ADR-199. A hunk whose path, or whose rename destination, lies below
// something that exists and is not a directory is refused on its hunk: no
// error, a failed hunk with mrw's words, nothing staged, nothing written.
func TestAPathBelowAFileIsRefusedOnItsHunk(t *testing.T) {
	const body = "a\nb\nc\n"
	p := filepath.FromSlash
	op := func(path, o string, b ...string) Input { return Input{Path: p(path), Op: o, Body: b, Lines: -1} }
	line := func(path, o string, s, e int, b ...string) Input {
		return Input{Path: p(path), Op: o, Start: s, End: e, Body: b, Lines: -1}
	}
	files := map[string]string{"d/f.txt": body, "d/g.txt": body}
	const aFile = "is a file, not a directory"
	words := func(path, anc, what string) string { return path + " cannot be written: " + anc + " " + what }
	link := func(target, name string) func(*testing.T, string) {
		return func(t *testing.T, root string) {
			if err := os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, filepath.Join(root, name)); err != nil {
				t.Skipf("links unavailable here: %v", err)
			}
		}
	}

	type refused struct {
		name  string
		setup func(t *testing.T, root string)
		plan  []Input
		want  string // the one failed hunk's reason
		// at is the plan index of the hunk that fails; every other is skipped.
		at int
	}
	below := "d/f.txt/x"
	cases := []refused{
		{name: "create one level below a file", plan: []Input{op(below, "create", "x")}, want: words(below, "d/f.txt", aFile)},
		{name: "create two levels below a file", plan: []Input{op("d/f.txt/sub/y.txt", "create", "x")}, want: words("d/f.txt/sub/y.txt", "d/f.txt", aFile)},
		{name: "replace below a file", plan: []Input{line(below, "replace", 1, 1, "x")}, want: words(below, "d/f.txt", aFile)},
		{name: "insert-after below a file", plan: []Input{line(below, "insert-after", 1, 0, "x")}, want: words(below, "d/f.txt", aFile)},
		{name: "insert-before below a file", plan: []Input{line(below, "insert-before", 1, 0, "x")}, want: words(below, "d/f.txt", aFile)},
		{name: "delete below a file", plan: []Input{line(below, "delete", 1, 1)}, want: words(below, "d/f.txt", aFile)},
		{name: "unlink below a file", plan: []Input{op(below, "unlink")}, want: words(below, "d/f.txt", aFile)},
		{name: "a rename whose source is below a file", plan: []Input{op(below, "rename", "d/n.txt")}, want: words(below, "d/f.txt", aFile)},
		{name: "a rename to a destination below a file", plan: []Input{op("d/g.txt", "rename", below)}, want: words(below, "d/f.txt", aFile)},
		{name: "a rename to a destination below a file, beside an edit of its source",
			plan: []Input{line("d/g.txt", "replace", 1, 1, "x"), op("d/g.txt", "rename", below)}, at: 1, want: words(below, "d/f.txt", aFile)},
		{name: "an edit beside a create below a file: the edit is skipped",
			plan: []Input{line("d/g.txt", "replace", 1, 1, "x"), op(below, "create", "x")}, at: 1, want: words(below, "d/f.txt", aFile)},
		{name: "an unlink of the file beside a create below it",
			plan: []Input{op("d/f.txt", "unlink"), op(below, "create", "x")}, at: 1, want: words(below, "d/f.txt", aFile)},
		// A link to a directory is walked through, and the file below it is
		// the obstruction, named by the plan's own path.
		{name: "create below a file that is under a link to a directory", setup: func(t *testing.T, root string) {
			write(t, root, "d/real/f", body)
			link("real", "d/ld")(t, root)
		}, plan: []Input{op("d/ld/f/x", "create", "x")}, want: words("d/ld/f/x", "d/ld/f", aFile)},
		{name: "the shortest obstruction is named", setup: link("f.txt", "d/lf"),
			plan: []Input{op("d/f.txt/lf/x", "create", "x")}, want: words("d/f.txt/lf/x", "d/f.txt", aFile)},
	}
	run := func(t *testing.T, c refused) {
		root := t.TempDir()
		for n, b := range files {
			write(t, root, n, b)
		}
		if c.setup != nil {
			c.setup(t, root)
		}
		noStaging(t)
		before := snapshot(t, root)
		in := append([]Input(nil), c.plan...)
		for i := range in {
			in[i].Index, in[i].SrcLine = i, i+1
		}
		res, err := Apply(root, in, Options{})
		if err != nil {
			t.Fatalf("Apply returned an error, want a failed hunk: %v", err)
		}
		if res.Applied || res.Failed != 1 {
			t.Fatalf("Applied=%v Failed=%d, want a refusal of one hunk: %+v", res.Applied, res.Failed, res.Hunks)
		}
		for i, h := range res.Hunks {
			if i == c.at {
				if h.Status != StatusFailed || h.Reason != c.want {
					t.Errorf("hunk %d: %s %q, want failed %q", i, h.Status, h.Reason, c.want)
				}
			} else if h.Status != StatusSkipped {
				t.Errorf("hunk %d: %s %q, want skipped", i, h.Status, h.Reason)
			}
		}
		if after := snapshot(t, root); !reflect.DeepEqual(before, after) {
			t.Errorf("a refused plan changed the tree:\nbefore %v\nafter  %v", before, after)
		}
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { run(t, c) })
	}

	// Windows reports a path below a regular file as not found, so the load
	// succeeds with existed=false; a test drives that answer through loadFn
	// while the tree still holds d/f.txt as a file.
	t.Run("a file that reports as missing below a file", func(t *testing.T) {
		for _, c := range []refused{
			{name: "create", plan: []Input{op(below, "create", "x")}, want: words(below, "d/f.txt", aFile)},
			{name: "replace", plan: []Input{line(below, "replace", 1, 1, "x")}, want: words(below, "d/f.txt", aFile)},
			{name: "unlink", plan: []Input{op(below, "unlink")}, want: words(below, "d/f.txt", aFile)},
		} {
			t.Run(c.name, func(t *testing.T) {
				real := loadFn
				t.Cleanup(func() { loadFn = real })
				loadFn = func(path string) (text, bool, error) {
					if strings.HasSuffix(path, filepath.FromSlash(below)) {
						return text{}, false, nil
					}
					return real(path)
				}
				run(t, c)
			})
		}
	})

	applied := func(t *testing.T, name string, setup func(*testing.T, string), plan Input, made string) {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			for n, b := range files {
				write(t, root, n, b)
			}
			if err := os.Mkdir(filepath.Join(root, "d", "real"), 0o755); err != nil {
				t.Fatal(err)
			}
			if setup != nil {
				setup(t, root)
			}
			plan.Index, plan.SrcLine = 0, 1
			res, err := Apply(root, []Input{plan}, Options{})
			if err != nil || !res.Applied || res.Failed != 0 {
				t.Fatalf("the plan did not apply: %v %+v", err, res)
			}
			if got := read(t, root, made); got != "x\n" {
				t.Errorf("%s = %q, want %q", made, got, "x\n")
			}
		})
	}
	applied(t, "kept: create below a directory that does not exist yet", nil, op("d/new/x.txt", "create", "x"), "d/new/x.txt")
	applied(t, "kept: create below a directory that exists", nil, op("d/x.txt", "create", "x"), "d/x.txt")
	applied(t, "kept: create below a link to a directory", link("real", "d/ld"), op("d/ld/x.txt", "create", "x"), "d/real/x.txt")
	// The parent is missing here, so the prefixes are walked and the link is
	// judged by where it leads: a check on the link itself would refuse it.
	applied(t, "kept: create below a missing directory under a link to a directory", link("real", "d/ld"), op("d/ld/new/x.txt", "create", "x"), "d/real/new/x.txt")
	// A link is never the obstruction. One that does not lead to a directory
	// is left to what happened before this record (an error from the load or
	// from staging), because where a link leads is told differently on each
	// platform: on Windows a link to nothing two levels deep answers "path not
	// found", which Go also reads as "not a directory", and a build that went
	// by that refused a plan that applies (the go-reviewer pass on #646).
	for _, c := range []struct {
		name  string
		setup func(*testing.T, string)
		path  string
	}{
		{"a link to a file", link("f.txt", "d/lf"), "d/lf/x"},
		{"a link whose target is below a file", link(filepath.Join("f.txt", "sub"), "d/lq"), "d/lq/x"},
		{"a link to nothing", link("nowhere", "d/ln"), "d/ln/x"},
		{"a link to nothing two levels deep", link(filepath.Join("..", "gone", "sub"), "d/l2"), "d/l2/x"},
		{"a link loop", func(t *testing.T, root string) {
			link("lb", "d/la")(t, root)
			link("la", "d/lb")(t, root)
		}, "d/la/x"},
	} {
		t.Run("left as before: create below "+c.name, func(t *testing.T) {
			root := t.TempDir()
			for n, b := range files {
				write(t, root, n, b)
			}
			c.setup(t, root)
			in := op(c.path, "create", "x")
			in.Index, in.SrcLine = 0, 1
			res, _ := Apply(root, []Input{in}, Options{})
			for _, h := range res.Hunks {
				if strings.Contains(h.Reason, "cannot be written") {
					t.Errorf("a path below %s was refused by this record's rule: %q", c.name, h.Reason)
				}
			}
		})
	}
	// The two-level target made by the same plan: it applied before, and the
	// build that read Windows' answer as "not a directory" refused it there.
	t.Run("kept: create below a link to a two-level directory the same plan makes", func(t *testing.T) {
		root := t.TempDir()
		for n, b := range files {
			write(t, root, n, b)
		}
		link(filepath.Join("..", "newdir", "sub"), "d/ln")(t, root)
		in := []Input{op("newdir/sub/y.txt", "create", "y"), op("d/ln/x.txt", "create", "x")}
		for i := range in {
			in[i].Index, in[i].SrcLine = i, i+1
		}
		res, err := Apply(root, in, Options{})
		if err != nil || !res.Applied || res.Failed != 0 {
			t.Fatalf("the plan did not apply: %v %+v", err, res.Hunks)
		}
		if got := read(t, root, "newdir/sub/x.txt"); got != "x\n" {
			t.Errorf("newdir/sub/x.txt = %q, want %q", got, "x\n")
		}
	})
	// A link to nothing is not an obstruction: the same plan may make the
	// directory it leads to, and that plan applied before this record (the
	// go-reviewer pass on the first version, which refused it).
	t.Run("kept: create below a link to a directory the same plan makes", func(t *testing.T) {
		root := t.TempDir()
		for n, b := range files {
			write(t, root, n, b)
		}
		link(filepath.Join("..", "newdir"), "d/ln")(t, root)
		in := []Input{op("newdir/y.txt", "create", "y"), op("d/ln/x.txt", "create", "x")}
		for i := range in {
			in[i].Index, in[i].SrcLine = i, i+1
		}
		res, err := Apply(root, in, Options{})
		if err != nil || !res.Applied || res.Failed != 0 {
			t.Fatalf("the plan did not apply: %v %+v", err, res.Hunks)
		}
		if got := read(t, root, "newdir/x.txt"); got != "x\n" {
			t.Errorf("newdir/x.txt = %q, want %q", got, "x\n")
		}
	})

	// A prefix that leaves the root is the boundary's to refuse, in its own
	// words, and nothing is made or changed outside.
	for _, c := range []struct{ name, target string }{
		{"a link to a directory outside the root", ""},
		{"a link to a file outside the root", "file.txt"},
	} {
		t.Run("kept: below "+c.name, func(t *testing.T) {
			root, outside := t.TempDir(), t.TempDir()
			write(t, root, "d/f.txt", body)
			write(t, outside, "file.txt", body)
			target := filepath.Join(outside, c.target)
			if err := os.Symlink(target, filepath.Join(root, "out")); err != nil {
				t.Skipf("links unavailable here: %v", err)
			}
			before := listing(t, outside)
			noStaging(t)
			res, err := Apply(root, []Input{{Path: p("out/x.txt"), Op: "create", Body: []string{"x"}, Lines: -1, SrcLine: 1}}, Options{})
			if err != nil {
				t.Fatalf("Apply returned an error: %v", err)
			}
			r := failedReason(res)
			if r == "" || strings.Contains(r, "cannot be written") || !strings.Contains(r, "only change files under the directory mrw was pointed at") {
				t.Errorf("reason %q, want the boundary's own words", r)
			}
			if after := listing(t, outside); !reflect.DeepEqual(before, after) {
				t.Errorf("the outside directory changed: %v -> %v", before, after)
			}
		})
	}

	t.Run("kept: two spellings of one file below a file", func(t *testing.T) {
		root := t.TempDir()
		for n, b := range files {
			write(t, root, n, b)
		}
		// The clash is judged on the names, before any file is loaded, on every
		// filesystem: on one that folds case both hunks are below the file.
		noStaging(t)
		in := []Input{op("d/f.txt/x", "create", "x"), op("d/F.TXT/x", "create", "y")}
		for i := range in {
			in[i].Index, in[i].SrcLine = i, i+1
		}
		res, err := Apply(root, in, Options{})
		if err != nil {
			t.Fatal(err)
		}
		if r := res.Hunks[1].Reason; res.Hunks[1].Status != StatusFailed || !strings.Contains(r, "one spelling per plan") || strings.Contains(r, "cannot be written") {
			t.Errorf("second hunk: %s %q, want the case-clash refusal to win", res.Hunks[1].Status, r)
		}
	})
}
