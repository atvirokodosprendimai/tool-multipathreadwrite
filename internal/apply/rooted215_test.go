package apply

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ADR-215. A plan path that is rooted is refused for every op, as a failed hunk
// before any file is looked up: the sibling skips, nothing is written, no
// directory is made. The fixtures reach each branch the record names: a path
// whose joined target is missing (a create that would have built the chain), one
// whose joined target EXISTS and is licensed, a rename whose source is licensed
// and one whose source was never read, a directory spelling, and a case pair.

const rootedSentence = "is absolute, and every path in a plan is relative to the root"

// snapshot215 names every entry under dir and the bytes of each file, so a test
// can say a refused plan changed nothing, directories included.
func snapshot215(t *testing.T, dir string) map[string]string {
	t.Helper()
	got := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			got[p] = "<dir>"
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		got[p] = string(b)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestEveryPlanOpRefusesARootedPath(t *testing.T) {
	const body = "a\nb\nc\n"
	// fixture is the tree every case starts from: dd/zz.txt, src.txt and ok.txt
	// exist and have been read; outside is a file beside the root, not in it.
	fixture := func(t *testing.T) (root, outside string, seen map[string]Seen) {
		t.Helper()
		root = t.TempDir()
		for _, f := range []string{"dd/zz.txt", "src.txt", "ok.txt"} {
			write(t, root, f, body)
		}
		outDir := t.TempDir()
		write(t, outDir, "out.txt", body)
		outside = filepath.Join(outDir, "out.txt")
		seen = map[string]Seen{}
		for _, f := range []string{"dd/zz.txt", "src.txt", "ok.txt"} {
			seen[filepath.FromSlash(f)] = Seen{SHA: shaOfFile(t, root, f), Spans: [][2]int{{1, 3}}}
		}
		return root, outside, seen
	}
	edit := func(p, op string) Input {
		in := Input{Path: p, Start: 1, End: 1, Op: op, Body: []string{"X"}, Lines: -1}
		if op == "delete" {
			in.Body = nil
		}
		return in
	}
	pathOp := func(p, op string, b ...string) Input { return Input{Path: p, Op: op, Body: b, Lines: -1} }
	create := func(p string) Input { return Input{Path: p, Op: "create", Body: []string{"X"}, Lines: -1} }
	sibling := Input{Path: "ok.txt", Start: 1, End: 1, Op: "replace", Body: []string{"S"}, Lines: -1}

	type tc struct {
		name string
		in   func(root, outside string) []Input // the rooted hunks; the sibling follows
		// rooted lists the positions in in that must fail; every other hunk skips.
		rooted []int
		// unread leaves src.txt out of the ledger.
		unread bool
		// nested is a file at ROOT/<root>/x, licensed under the rooted spelling,
		// which the old join reached and applied to.
		nested bool
	}
	var cases []tc
	for _, op := range []string{"replace", "insert-before", "insert-after", "delete", "unlink"} {
		op := op
		mk := func(p string) Input {
			if op == "unlink" {
				return pathOp(p, op)
			}
			return edit(p, op)
		}
		cases = append(cases,
			tc{name: op + " /dd/zz.txt", in: func(string, string) []Input { return []Input{mk("/dd/zz.txt")} }, rooted: []int{0}},
			tc{name: op + " an absolute path inside the root", in: func(root, _ string) []Input {
				return []Input{mk(filepath.Join(root, "dd", "zz.txt"))}
			}, rooted: []int{0}},
			tc{name: op + " an absolute path outside the root", in: func(_, out string) []Input { return []Input{mk(out)} }, rooted: []int{0}},
		)
	}
	cases = append(cases,
		tc{name: "rename source /dd/zz.txt", in: func(string, string) []Input { return []Input{pathOp("/dd/zz.txt", "rename", "moved.txt")} }, rooted: []int{0}},
		tc{name: "rename of an absolute source inside the root", in: func(root, _ string) []Input {
			return []Input{pathOp(filepath.Join(root, "src.txt"), "rename", "moved.txt")}
		}, rooted: []int{0}},
		tc{name: "create /new/x, no nested target", in: func(string, string) []Input { return []Input{create("/new/x")} }, rooted: []int{0}},
		tc{name: "create at an absolute path inside the root", in: func(root, _ string) []Input {
			return []Input{create(filepath.Join(root, "newdir", "x"))}
		}, rooted: []int{0}},
		tc{name: "rename of a read source to /new/x", in: func(string, string) []Input { return []Input{pathOp("src.txt", "rename", "/new/x")} }, rooted: []int{0}},
		tc{name: "rename of a read source to an absolute path inside the root", in: func(root, _ string) []Input {
			return []Input{pathOp("src.txt", "rename", filepath.Join(root, "moved.txt"))}
		}, rooted: []int{0}},
		tc{name: "rename of an unread source to /new/x", in: func(string, string) []Input { return []Input{pathOp("src.txt", "rename", "/new/x")} }, rooted: []int{0}, unread: true},
		tc{name: "a directory spelling /dd/x/", in: func(string, string) []Input { return []Input{create("/dd/x/")} }, rooted: []int{0}},
		tc{name: "a case pair /new/x and /NEW/x", in: func(string, string) []Input { return []Input{create("/new/x"), create("/NEW/x")} }, rooted: []int{0, 1}},
	)
	if runtime.GOOS != "windows" {
		cases = append(cases, tc{name: "a nested target that exists and is licensed", in: func(root, _ string) []Input {
			return []Input{edit(filepath.Join(root, "x"), "replace")}
		}, rooted: []int{0}, nested: true})
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root, outside, seen := fixture(t)
			in := c.in(root, outside)
			if c.nested {
				// ROOT/<root>/x: what `join(ROOT, <root>/x)` named before ADR-215.
				write(t, root, filepath.Join(strings.TrimPrefix(root, string(filepath.Separator)), "x"), body)
				key := filepath.Join(root, "x")
				seen[key] = Seen{SHA: shaOfFile(t, root, filepath.Join(strings.TrimPrefix(root, string(filepath.Separator)), "x")), Spans: [][2]int{{1, 3}}}
			}
			if c.unread {
				delete(seen, "src.txt")
			}
			in = append(in, sibling)
			for n := range in {
				in[n].Index, in[n].SrcLine = n, n+1
			}
			before, outBefore := snapshot215(t, root), snapshot215(t, filepath.Dir(outside))

			res, err := Apply(root, in, Options{Seen: seen})
			if err != nil {
				t.Fatalf("a refused plan is a verdict, not an error: %v", err)
			}
			if res.Applied || res.Failed != len(c.rooted) {
				t.Fatalf("applied %v, %d failed, want a refusal of %d hunk(s): %+v", res.Applied, res.Failed, len(c.rooted), res.Hunks)
			}
			isRooted := map[int]bool{}
			for _, n := range c.rooted {
				isRooted[n] = true
			}
			for n, h := range res.Hunks {
				if isRooted[n] {
					if h.Status != StatusFailed || !strings.Contains(h.Reason, rootedSentence) {
						t.Errorf("hunk %d is %s %q, want failed with %q", n, h.Status, h.Reason, rootedSentence)
					}
					if strings.Contains(h.Reason, "mrw looked for") || strings.Contains(h.Reason, "has not been read") {
						t.Errorf("hunk %d reason %q reaches a lookup", n, h.Reason)
					}
				} else if h.Status != StatusSkipped {
					t.Errorf("hunk %d is %s, want skipped", n, h.Status)
				}
			}
			if after := snapshot215(t, root); !equalTrees215(before, after) {
				t.Errorf("the root changed:\nbefore %v\nafter  %v", before, after)
			}
			if after := snapshot215(t, filepath.Dir(outside)); !equalTrees215(outBefore, after) {
				t.Errorf("the directory outside the root changed")
			}
		})
	}

	// A rooted hunk beside one that would have set the load error (a directory
	// named as a file) is an exit-1 outcome: nothing is loaded.
	t.Run("a rooted hunk beside a hunk that would not load", func(t *testing.T) {
		root, _, seen := fixture(t)
		in := []Input{create("/new/x"), edit("dd", "replace")}
		for n := range in {
			in[n].Index, in[n].SrcLine = n, n+1
		}
		res, err := Apply(root, in, Options{Seen: seen})
		if err != nil {
			t.Fatalf("err %v: the preflight loads nothing, so no load error exists", err)
		}
		if res.Hunks[0].Status != StatusFailed || res.Hunks[1].Status != StatusSkipped {
			t.Errorf("statuses %s, %s, want failed, skipped", res.Hunks[0].Status, res.Hunks[1].Status)
		}
	})

	// What the message holds: a spelling only where the meaning is clear.
	t.Run("the message", func(t *testing.T) {
		root, _, seen := fixture(t)
		const inside = "for the file inside the root write "
		for _, m := range []struct {
			name string
			in   func() Input
			want string // empty: no suggested spelling at all
		}{
			{"inside the root", func() Input { return edit(filepath.Join(root, "dd", "zz.txt"), "replace") }, inside + "dd/zz.txt"},
			{"a leading @ is spelled ./@", func() Input { return edit(filepath.Join(root, "@x"), "replace") }, inside + "./@x"},
			{"a space is double-quoted", func() Input { return edit(filepath.Join(root, "a b.txt"), "replace") }, inside + `"a b.txt"`},
			{"a rename destination is the literal body line", func() Input { return pathOp("src.txt", "rename", filepath.Join(root, "a b.txt")) }, inside + "a b.txt"},
			{"/etc/hosts names none", func() Input { return edit("/etc/hosts", "replace") }, ""},
			{"/../x names none", func() Input { return edit("/../x", "replace") }, ""},
			{"/ names none", func() Input { return edit("/", "replace") }, ""},
			{"the root itself names none", func() Input { return edit(root, "replace") }, ""},
		} {
			in := m.in()
			in.Index, in.SrcLine = 0, 1
			res, err := Apply(root, []Input{in}, Options{Seen: seen})
			if err != nil || len(res.Hunks) != 1 {
				t.Fatalf("%s: %v %+v", m.name, err, res.Hunks)
			}
			r := res.Hunks[0].Reason
			if !strings.Contains(r, rootedSentence) {
				t.Errorf("%s: reason %q lacks %q", m.name, r, rootedSentence)
			}
			if m.want != "" && !strings.Contains(r, m.want) {
				t.Errorf("%s: reason %q lacks %q", m.name, r, m.want)
			}
			if m.want == "" && (strings.Contains(r, "write ") || !strings.Contains(r, "name the file relative to the root, without the leading separator or drive")) {
				t.Errorf("%s: reason %q suggests a spelling, or does not describe", m.name, r)
			}
		}
	})
}

func equalTrees215(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || w != v {
			return false
		}
	}
	return true
}
