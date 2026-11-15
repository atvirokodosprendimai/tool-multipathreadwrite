package apply

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// ADR-198. A refusal reason spells a plan's path as its row does: with "/",
// wherever mrw's own words put a cleaned root-relative path. Where the
// platform's separator is "/" the test sets the reason separator to `\` and
// names its files with a backslash, which is then one file name (`d\f.txt`);
// on Windows it is the real separator and `d` is a real directory. Either way
// the expected text is d/f.txt, and the reason is read from the Result Apply
// returned, before any receipt converts it.
func TestAReasonSpellsAPlansPathAsItsRowDoes(t *testing.T) {
	withSep := func(t *testing.T, r rune) {
		t.Helper()
		prev := reasonSep
		reasonSep = r
		t.Cleanup(func() { reasonSep = prev })
	}
	if filepath.Separator == '/' {
		withSep(t, '\\')
	}
	// nat is the native spelling of a name written with "/": a name holding
	// backslashes here, a path with directories on Windows.
	nat := func(p string) string { return strings.ReplaceAll(p, "/", `\`) }
	// under puts leaf in the directory dir with the platform's own separator,
	// so a `/` directory holds a name with a backslash where "/" is the separator.
	under := func(dir, leaf string) string { return dir + string(filepath.Separator) + leaf }
	f, g, h := nat("d/f.txt"), nat("d/g.txt"), nat("d/h.txt")
	const body = "a\nb\nc\n"

	ed := func(p string, s, e int) Input {
		return Input{Path: p, Start: s, End: e, Op: "replace", Body: []string{"x"}, Lines: -1}
	}
	op := func(p, o string, b ...string) Input { return Input{Path: p, Op: o, Body: b, Lines: -1} }
	pat := func(p, start string) Input {
		i := ed(p, 0, 0)
		i.StartPat = regexp.MustCompile(start)
		return i
	}
	from := func(dest, src string, s, e int) Input {
		return Input{Path: dest, Op: "create", From: src, FromStart: s, FromEnd: e, Lines: -1}
	}
	fixed := func(in ...Input) func(string) ([]Input, Options) {
		return func(string) ([]Input, Options) { return in, Options{} }
	}
	// ledger builds a read ledger from what the files are on disk.
	ledger := func(t *testing.T, root string, spans map[string][][2]int) map[string]Seen {
		l := map[string]Seen{}
		for n, sp := range spans {
			l[n] = Seen{SHA: shaOfFile(t, root, n), Spans: sp}
		}
		return l
	}
	hardLink := func(from, to string) func(*testing.T, string) {
		return func(t *testing.T, root string) {
			if err := os.Link(filepath.Join(root, from), filepath.Join(root, to)); err != nil {
				t.Skipf("hard links unavailable here: %v", err)
			}
		}
	}
	foldsCase := func(t *testing.T, root string) {
		if !caseInsensitiveFS(t, root) {
			t.Skip("this filesystem does not fold case: the respelling refusals are not reachable here")
		}
	}

	type site struct {
		name  string
		files map[string]string // native name -> content
		setup func(t *testing.T, root string)
		plan  func(root string) ([]Input, Options)
		// viaErr: Apply returns an error, and its text carries the wanted
		// words as the reason does.
		viaErr bool
		// lead is the hunk's own path a reason may begin with, which a
		// receipt converts and Apply does not.
		lead   string
		want   []string
		absent []string // checked after lead is cut from the start
	}
	one := map[string]string{f: body}
	two := map[string]string{f: body, g: body}
	sites := []site{
		{name: "unread lines", files: one, plan: func(root string) ([]Input, Options) {
			return []Input{ed(f, 1, 1)}, Options{Seen: ledger(t, root, map[string][][2]int{f: {{3, 3}}})}
		}, want: []string{"of d/f.txt has not been read"}, absent: []string{f}},
		{name: "unread file, advice for the CLI", files: one, plan: func(string) ([]Input, Options) {
			return []Input{ed(f, 1, 1)}, Options{Seen: map[string]Seen{}}
		}, lead: f, want: []string{"Run `mrw read d/f.txt` first"}, absent: []string{f}},
		{name: "unread file, advice for mrw_write", files: one, plan: func(string) ([]Input, Options) {
			return []Input{ed(f, 1, 1)}, Options{Seen: map[string]Seen{}, NoForce: true}
		}, lead: f, want: []string{"Read d/f.txt with mrw_read first"}, absent: []string{f}},
		{name: "an unlink beside an edit", files: one, plan: fixed(op(f, "unlink"), ed(f, 1, 1)),
			want: []string{"unlink/rename cannot mix with other hunks on d/f.txt"}, absent: []string{f}},
		{name: "a create of a file that exists", files: one, plan: func(string) ([]Input, Options) {
			return []Input{op(f, "create", "x")}, Options{Seen: map[string]Seen{}}
		}, want: []string{"create: d/f.txt already exists"}, absent: []string{f}},
		{name: "a create beside another hunk", files: one, plan: fixed(op(f, "create", "x"), ed(f, 1, 1)),
			want: []string{"create: d/f.txt already exists"}, absent: []string{f}},
		{name: "sha= on a path that does not exist", plan: func(string) ([]Input, Options) {
			i := ed(g, 1, 1)
			i.SHA = "deadbeef"
			return []Input{i}, Options{}
		}, want: []string{"given but d/g.txt does not exist"}, absent: []string{g}},
		{name: "sha= on an unlink of a path that does not exist", plan: func(string) ([]Input, Options) {
			i := op(g, "unlink")
			i.SHA = "deadbeef"
			return []Input{i}, Options{}
		}, want: []string{"given but d/g.txt does not exist"}, absent: []string{g}},
		{name: "fewer matches than occurrence=", files: map[string]string{f: "x\nx\ny\n"}, plan: func(string) ([]Input, Options) {
			i := pat(f, "x")
			i.Occurrence = 5
			return []Input{i}, Options{}
		}, want: []string{"matched 2 line(s) in d/f.txt (lines 1, 2)"}, absent: []string{f}},
		{name: "unread matches before occurrence=", files: map[string]string{f: "x\nx\ny\n"}, plan: func(root string) ([]Input, Options) {
			i := pat(f, "x")
			i.Occurrence = 2
			return []Input{i}, Options{Seen: ledger(t, root, map[string][][2]int{f: {{2, 2}}})}
		}, want: []string{"lines 1 of d/f.txt before occurrence 2"}, absent: []string{f}},
		{name: "a pattern that matches no line", files: one, plan: fixed(pat(f, "zzz")),
			want: []string{"matched no line in d/f.txt"}, absent: []string{f}},
		{name: "a start pattern that matches many lines", files: map[string]string{f: "x\nx\ny\n"}, plan: fixed(pat(f, "x")),
			want: []string{"matched 2 lines in d/f.txt (lines 1, 2)"}, absent: []string{f}},
		{name: "an end pattern that matches no line", files: one, plan: func(string) ([]Input, Options) {
			i := pat(f, "^b$")
			i.EndPat = regexp.MustCompile("zzz")
			return []Input{i}, Options{}
		}, want: []string{"end pattern zzz matched no line in d/f.txt"}, absent: []string{f}},
		{name: "an end pattern that matches only above the start", files: one, plan: func(string) ([]Input, Options) {
			i := pat(f, "^c$")
			i.EndPat = regexp.MustCompile("^a$")
			return []Input{i}, Options{}
		}, want: []string{"matched only above the start line 3 (lines 1) in d/f.txt"}, absent: []string{f}},
		{name: "an anchor in no line", files: one, plan: func(string) ([]Input, Options) {
			i := ed(f, 2, 2)
			i.Anchor = "nowhere"
			return []Input{i}, Options{}
		}, want: []string{"; no line of d/f.txt holds it"}, absent: []string{f}},
		{name: "two spellings of one existing file", files: one, setup: hardLink(f, g), plan: fixed(ed(f, 1, 1), ed(g, 1, 1)),
			lead: f, want: []string{"names the same file as d/"}, absent: []string{"as " + nat("d/")}},
		{name: "two names that differ only in case, one a rename's destination", files: one, plan: fixed(
			op(f, "rename", nat("d/Q.txt")), op(nat("d/q.txt"), "create", "x")),
			want: []string{"d/q.txt may name the same file as d/Q.txt"}, absent: []string{nat("d/")}},
		// foldClashes runs before an absolute destination is refused, so it
		// can hold a rooted name: that one stays as the caller typed it.
		{name: "two rooted rename destinations that differ only in case", files: two, plan: fixed(
			op(f, "rename", string(filepath.Separator)+nat("missing/Q.txt")), op(g, "rename", string(filepath.Separator)+nat("missing/q.txt"))),
			want: []string{string(filepath.Separator) + nat("missing/q.txt") + " may name the same file as " + string(filepath.Separator) + nat("missing/Q.txt")}},
		{name: "a rename onto the source", files: one, plan: fixed(op(f, "rename", f)),
			want: []string{"rename dest d/f.txt is the source"}, absent: []string{f}},
		{name: "a rename dest named twice", files: two, plan: fixed(op(f, "rename", nat("d/t.txt")), op(g, "rename", nat("d/t.txt"))),
			want: []string{"rename dest d/t.txt is named by more than one hunk"}, absent: []string{nat("d/t")}},
		{name: "a rename dest also written", files: one, plan: fixed(op(f, "rename", g), op(g, "create", "x")),
			want: []string{"rename dest d/g.txt is also written by another hunk"}, absent: []string{g}},
		{name: "a rename dest that nests with another dest", files: two, plan: func(string) ([]Input, Options) {
			return []Input{op(f, "rename", nat("a/b")), op(g, "rename", under(nat("a/b"), "c.txt"))}, Options{}
		}, want: []string{"nests with rename dest a/b"}, absent: []string{nat("a/")}},
		{name: "a rename dest that nests with a written file", files: one, plan: func(string) ([]Input, Options) {
			return []Input{op(f, "rename", nat("a/b")), op(under(nat("a/b"), "c.txt"), "create", "x")}, Options{}
		}, want: []string{"rename dest a/b nests with a/b/c.txt written by another hunk"}, absent: []string{nat("a/")}},
		{name: "a rename dest that exists", files: two, plan: fixed(op(f, "rename", g)),
			want: []string{"rename dest d/g.txt already exists"}, absent: []string{g}},
		// ADR-066's fixture (a 300-byte leaf, past NAME_MAX on every platform,
		// under a directory that exists): a destination whose Lstat fails for
		// a reason mrw has no words of its own for. A path under a file is
		// ADR-199's, and refused in its words.
		{name: "a rename dest that cannot be used", files: map[string]string{f: body, g: body, under(nat("a/b"), "k.txt"): body},
			plan: fixed(op(g, "rename", under(nat("a/b"), long))),
			want: []string{"rename dest a/b/" + long + " cannot be used"}, absent: []string{"dest " + nat("a/")}},
		{name: "a rename dest spelled as a directory", files: map[string]string{"f0.txt": body}, plan: func(string) ([]Input, Options) {
			return []Input{op("f0.txt", "rename", nat("e/s")+string(filepath.Separator))}, Options{}
		}, want: []string{"rename dest " + nat("e/s") + string(filepath.Separator), "such as e/s/f0.txt"}},
		{name: "a rename that respells its directory", setup: foldsCase, files: map[string]string{under(nat("a/b"), "x.txt"): body},
			plan: fixed(op(under(nat("a/b"), "x.txt"), "rename", under(nat("A/B"), "X.txt"))),
			want: []string{"rename dest A/B/X.txt respells the directory a/b"}, absent: []string{nat("A/")}},
		{name: "a rename onto the source under another spelling of its directory", setup: foldsCase, files: map[string]string{under(nat("a/b"), "x.txt"): body},
			plan: fixed(op(under(nat("a/b"), "x.txt"), "rename", under(nat("A/B"), "x.txt"))),
			want: []string{"rename dest A/B/x.txt is the source"}, absent: []string{nat("A/")}},
		{name: "a rename whose source is not spelled as its directory lists it", setup: foldsCase, files: map[string]string{under(nat("a/b"), "x.txt"): body},
			plan: fixed(op(under(nat("a/b"), "X.txt"), "rename", under(nat("a/b"), "x.txt"))),
			want: []string{"rename source a/b/X.txt is not spelled that way"}, absent: []string{nat("a/")}},
		{name: "from= a source that does not exist", files: one, plan: fixed(from(g, h, 1, 1)),
			want: []string{"from=d/h.txt does not exist"}, absent: []string{h}},
		{name: "from= a directory", files: one, setup: func(t *testing.T, root string) {
			if err := os.MkdirAll(filepath.Join(root, nat("d/dir")), 0o755); err != nil {
				t.Fatal(err)
			}
		}, plan: fixed(from(g, nat("d/dir"), 1, 1)), want: []string{"from=d/dir is a directory"}, absent: []string{nat("d/dir")}},
		{name: "from= an unread source", files: one, plan: func(string) ([]Input, Options) {
			return []Input{from(g, f, 1, 1)}, Options{Seen: map[string]Seen{}}
		}, lead: f, want: []string{"Run `mrw read d/f.txt:1-1` first"}, absent: []string{f}},
		{name: "from= lines that were not shown", files: one, plan: func(root string) ([]Input, Options) {
			return []Input{from(g, f, 1, 1)}, Options{Seen: ledger(t, root, map[string][][2]int{f: {}})}
		}, want: []string{"of d/f.txt have not been shown"}, absent: []string{f}},
		{name: "from= a span that runs past the source", files: one, plan: fixed(from(g, f, 1, 99)),
			want: []string{"from=d/f.txt: lines 1-99 run past"}, absent: []string{f}},
		{name: "from= a source in another encoding", files: map[string]string{f: "\xff\xfea\x00\n\x00"}, plan: fixed(from(g, f, 1, 1)),
			want: []string{"from=d/f.txt ", "mrw copies UTF-8 text line by line"}, absent: []string{f}},
		{name: "a rename dest that appears before commit, and the files already written", files: map[string]string{f: body, h: body},
			plan: fixed(ed(h, 1, 1), op(f, "rename", g)), viaErr: true, setup: func(t *testing.T, root string) {
				real := commitRenameFn
				t.Cleanup(func() { commitRenameFn = real })
				commitRenameFn = func(tr *tree, from, to string) error {
					err := real(tr, from, to)
					if filepath.Base(to) == filepath.Base(h) {
						write(t, root, g, "newcomer\n")
					}
					return err
				}
			}, want: []string{"rename dest d/g.txt appeared before commit", "ALREADY WRITTEN: d/h.txt"}, absent: []string{g, h}},
		{name: "a rename dest under a link to nothing", files: one, setup: func(t *testing.T, root string) {
			// The link's own parent: the root where "/" is the separator (the
			// name is then `a\l`), the directory a on Windows.
			if err := os.MkdirAll(filepath.Dir(filepath.Join(root, nat("a/l"))), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("nowhere", filepath.Join(root, nat("a/l"))); err != nil {
				t.Skipf("links unavailable here: %v", err)
			}
		}, plan: fixed(op(f, "rename", under(nat("a/l"), "x.txt"))),
			want: []string{"rename dest a/l/x.txt is under"}, absent: []string{"dest " + nat("a/")}},
		// apply.go's `rename dest D: ERR`: the destination's directory leaves
		// the root after validation judged it (ADR-106's swap), so the staging
		// check refuses it. The leaf holds the separator; sub is a real directory.
		{name: "a rename dest whose directory left the root after validation", files: map[string]string{"x.txt": body, "b.txt": body, filepath.Join("sub", "keep.txt"): body},
			setup: func(t *testing.T, root string) { swapAfterFirstStage(t, root, t.TempDir()) },
			plan: func(string) ([]Input, Options) {
				return []Input{ed("x.txt", 1, 1), op("b.txt", "rename", under("sub", nat("deep/b.txt")))}, Options{Force: true}
			}, want: []string{"rename dest sub/deep/b.txt: "}, absent: []string{nat("deep/b")}},
	}
	for _, c := range sites {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			for n, b := range c.files {
				write(t, root, n, b)
			}
			if c.setup != nil {
				c.setup(t, root)
			}
			in, opt := c.plan(root)
			for i := range in {
				in[i].Index, in[i].SrcLine = i, i+1
			}
			res, err := Apply(root, in, opt)
			if (err != nil) != c.viaErr {
				t.Fatalf("err = %v, want an error: %v", err, c.viaErr)
			}
			var reasons []string
			for _, hr := range res.Hunks {
				if hr.Status == StatusFailed {
					reasons = append(reasons, hr.Reason)
				}
			}
			if len(reasons) == 0 {
				t.Fatalf("no hunk failed: err %v, hunks %+v", err, res.Hunks)
			}
			all := strings.Join(reasons, "\n")
			var cut []string
			for _, r := range reasons {
				cut = append(cut, strings.TrimPrefix(r, c.lead))
			}
			rest := strings.Join(cut, "\n")
			for _, w := range c.want {
				if !strings.Contains(all, w) {
					t.Errorf("reason %q lacks %q", all, w)
				}
				if c.viaErr && !strings.Contains(err.Error(), w) {
					t.Errorf("error %q lacks %q, which the reason holds", err, w)
				}
			}
			for _, a := range c.absent {
				if strings.Contains(rest, a) {
					t.Errorf("reason %q holds the platform's spelling %q", all, a)
				}
			}
		})
	}

	// Kept as the caller typed them, or as the system prints them.
	t.Run("kept: a path as typed, absolute paths, from=X", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), `r\oot`)
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatal(err)
		}
		write(t, root, "f0.txt", body)
		write(t, root, f, body)
		abs := filepath.Join(root, "x.txt")
		realRoot, err := filepath.EvalSymlinks(root)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range []struct {
			name string
			in   []Input
			want []string
		}{
			{"an absolute rename destination", []Input{op("f0.txt", "rename", abs)},
				[]string{abs + " is absolute", "mrw looked for " + abs}},
			{"an absolute path that does not exist", []Input{op("/nothere.txt", "unlink")},
				[]string{"mrw looked for " + realRoot}},
			{"from=X echoed as the header spelled it", []Input{{Path: g, Op: "create", Body: []string{"x"}, From: f, FromStart: 1, FromEnd: 1, Lines: -1}},
				[]string{"from=" + f + ":1 is the hunk's body"}},
			{"a from= span that is no source address", []Input{{Path: g, Op: "create", From: f, FromStart: 0, FromEnd: 1, Lines: -1}},
				[]string{"from=" + f + ": the span"}},
		} {
			c.in[0].Index, c.in[0].SrcLine = 0, 1
			res, err := Apply(root, c.in, Options{})
			if err != nil {
				t.Fatalf("%s: %v", c.name, err)
			}
			r := failedReason(res)
			if r == "" {
				t.Fatalf("%s: no hunk failed: %+v", c.name, res.Hunks)
			}
			for _, w := range c.want {
				if !strings.Contains(r, w) {
					t.Errorf("%s: reason %q lacks %q, which is kept as it was typed or printed", c.name, r, w)
				}
			}
		}
	})

	t.Run("kept: a read-only file's command keeps the platform's spelling", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, f, body)
		if err := os.Chmod(filepath.Join(root, f), 0o444); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(filepath.Join(root, f), 0o644) })
		res, err := Apply(root, []Input{ed(f, 1, 1)}, Options{})
		if err != nil {
			t.Fatal(err)
		}
		r := failedReason(res)
		fix := "chmod u+w "
		if runtime.GOOS == "windows" {
			fix = "attrib -r "
		}
		if !strings.HasPrefix(r, f+" is read-only") || !strings.HasSuffix(r, "clear the mark first: "+fix+f) {
			t.Errorf("reason %q, want it to begin with the plan's path as Apply has it and end with %q", r, fix+f)
		}
	})

	t.Run("kept: every path field keeps the platform's spelling", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, f, body)
		res, err := Apply(root, []Input{op(f, "rename", g)}, Options{})
		if err != nil || !res.Applied {
			t.Fatalf("the rename did not apply: %v %+v", err, res)
		}
		if res.Hunks[0].Path != f || res.Files[0].Path != f || res.Files[0].RenamedTo != g {
			t.Errorf("hunk path %q, file %q, renamed to %q: want %q, %q, %q", res.Hunks[0].Path, res.Files[0].Path, res.Files[0].RenamedTo, f, f, g)
		}
		write(t, root, h, body)
		i := ed(h, 2, 2)
		i.Anchor = "nowhere"
		res, err = Apply(root, []Input{i}, Options{})
		if err != nil || res.Applied {
			t.Fatalf("the anchor miss applied: %v %+v", err, res)
		}
		if res.Hunks[0].Path != h || res.Files[0].Path != h {
			t.Errorf("refused hunk path %q, file %q: want %q", res.Hunks[0].Path, res.Files[0].Path, h)
		}
	})

	t.Run("kept: with / as the separator a name holding a backslash is named as it is", func(t *testing.T) {
		withSep(t, '/')
		root := t.TempDir()
		write(t, root, `a\b.txt`, body)
		i := ed(`a\b.txt`, 2, 2)
		i.Anchor = "nowhere"
		res, err := Apply(root, []Input{i}, Options{})
		if err != nil {
			t.Fatal(err)
		}
		if r := failedReason(res); !strings.HasSuffix(r, `no line of a\b.txt holds it`) {
			t.Errorf("reason %q, want it to end %q", r, `no line of a\b.txt holds it`)
		}
	})
}
