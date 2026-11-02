package apply

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ADR-007: mrw says which path it looked for. A plan path that does not exist,
// where a file mrw served the caller ends in that path, names the served file:
// in the native-versus-MRW run 7 an agent's plan said parser_0.py for
// parsers/parser_0.py, which it had read, and two turns went on "does not
// exist". Names are whole path components, sorted, at most three; an entry whose
// file is gone, or that a write would refuse (.git, a link out of the root),
// names nothing (the go-reviewer pass of #592).
func TestAMissingPlanPathNamesTheServedFileItEndsLike(t *testing.T) {
	const tail = " — every path in a plan is relative to the root"
	type tc struct {
		name  string
		files []string
		seen  []string
		want  string // the reason after "parser_0.py does not exist", exactly
	}
	cases := []tc{
		{"served under a directory", []string{"parsers/parser_0.py", "parsers/xparser_0.py"},
			[]string{"parsers/parser_0.py", "parsers/xparser_0.py"}, "; you were served parsers/parser_0.py" + tail},
		{"four, sorted, three named", []string{"d/parser_0.py", "a/parser_0.py", "c/parser_0.py", "b/parser_0.py"},
			[]string{"d/parser_0.py", "a/parser_0.py", "c/parser_0.py", "b/parser_0.py"},
			"; you were served a/parser_0.py, b/parser_0.py, c/parser_0.py, …" + tail},
		{"nothing served", []string{"parsers/parser_0.py"}, nil, ""},
		{"served, since removed", []string{"parsers/parser_0.py"}, []string{"old/parser_0.py"}, ""},
		{"served from .git", []string{".git/parser_0.py"}, []string{".git/parser_0.py"}, ""},
	}
	if runtime.GOOS != "windows" {
		cases = append(cases, tc{"served through a link that now leaves the root", nil, []string{"lnk/parser_0.py"}, ""})
	}
	for _, c := range cases {
		root := t.TempDir()
		for _, f := range c.files {
			write(t, root, f, "a\nb\n")
		}
		if strings.HasPrefix(c.name, "served through a link") {
			out := t.TempDir()
			write(t, out, "parser_0.py", "a\n")
			if err := os.Symlink(out, filepath.Join(root, "lnk")); err != nil {
				t.Fatal(err)
			}
		}
		seen := map[string]Seen{}
		for _, s := range c.seen {
			seen[filepath.FromSlash(s)] = Seen{}
		}
		res, err := Apply(root, []Input{
			{Path: "parser_0.py", Start: 1, End: 1, Op: "replace", Body: []string{"x"}, Lines: -1, Index: 0},
		}, Options{Seen: seen})
		if err != nil {
			t.Fatal(err)
		}
		if r, want := res.Hunks[0].Reason, "parser_0.py does not exist"+c.want; res.Failed != 1 || r != want {
			t.Errorf("%s: failed=%d reason\n %q, want\n %q", c.name, res.Failed, r, want)
		}
	}
}

// The same hint on every refusal of a missing plan path: unlink, rename, a
// replace planned beside a rename, and a sha= guard on a line op and on an
// unlink name the served file too (the gap #592 first left; the go-reviewer
// pass of 14ce08d found the last two untested). The pair: a path nothing
// served ends in gets the bare refusal.
func TestEveryMissingPathRefusalNamesTheServedFile(t *testing.T) {
	const hint = "; you were served parsers/parser_0.py — every path in a plan is relative to the root"
	const missing = "parser_0.py does not exist"
	for _, c := range []struct {
		name string
		plan []Input
		at   int // the hunk whose reason carries the refusal
		bare string
	}{
		{"unlink", []Input{{Path: "parser_0.py", Op: "unlink", Lines: -1}}, 0, missing},
		{"rename", []Input{{Path: "parser_0.py", Op: "rename", Body: []string{"q.py"}, Lines: -1}}, 0, missing},
		{"a replace beside a rename", []Input{
			{Path: "parser_0.py", Start: 1, End: 1, Op: "replace", Body: []string{"x"}, Lines: -1, Index: 0},
			{Path: "parser_0.py", Op: "rename", Body: []string{"q.py"}, Lines: -1, Index: 1},
		}, 1, missing},
		{"sha= on a replace", []Input{{Path: "parser_0.py", Start: 1, End: 1, Op: "replace", Body: []string{"x"}, SHA: "abcd1234", Lines: -1}},
			0, "sha=abcd1234 given but parser_0.py does not exist"},
		{"sha= on an unlink", []Input{{Path: "parser_0.py", Op: "unlink", SHA: "abcd1234", Lines: -1}},
			0, "sha=abcd1234 given but parser_0.py does not exist"},
	} {
		for _, served := range []bool{true, false} {
			root := t.TempDir()
			write(t, root, "parsers/parser_0.py", "a\nb\n")
			seen := map[string]Seen{}
			if served {
				seen[filepath.FromSlash("parsers/parser_0.py")] = Seen{}
			}
			res, err := Apply(root, c.plan, Options{Seen: seen})
			if err != nil {
				t.Fatal(err)
			}
			want := c.bare
			if served {
				want += hint
			}
			if r := res.Hunks[c.at].Reason; res.Failed < 1 || r != want {
				t.Errorf("%s served=%v: failed=%d reason\n %q, want\n %q\n all: %+v", c.name, served, res.Failed, r, want, res.Hunks)
			}
		}
	}
}
