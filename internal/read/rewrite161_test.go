package read

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
)

// rwHit is the match of the nth (from 0) occurrence of old in data, rewritten
// to repl, with the fields ast-grep --json reports for it.
func rwHit(t *testing.T, data, old, repl string, nth int) RewriteHit {
	t.Helper()
	at := -1
	for i := 0; i <= nth; i++ {
		j := strings.Index(data[at+1:], old)
		if j < 0 {
			t.Fatalf("%q has no occurrence %d of %q", data, nth, old)
		}
		at += 1 + j
	}
	start, end := at, at+len(old)
	return RewriteHit{Text: &old, Replacement: &repl, Start: &start, End: &end}
}

func rwParse(t *testing.T, r *Rewrite) []plan.Hunk {
	t.Helper()
	hs, err := plan.Parse(strings.NewReader(r.Plan.String()))
	if err != nil {
		t.Fatalf("the plan does not parse: %v\n%s", err, r.Plan.String())
	}
	return hs
}

type rwWant struct {
	start, end int
	body       []string
	anchor     string
}

func rwCheck(t *testing.T, r *Rewrite, path, data string, want []rwWant) {
	t.Helper()
	hs := rwParse(t, r)
	if len(hs) != len(want) || r.Sites != len(want) {
		t.Fatalf("%d hunk(s), %d site(s), want %d:\n%s", len(hs), r.Sites, len(want), r.Plan.String())
	}
	sum := sha256.Sum256([]byte(data))
	for i, w := range want {
		h := hs[i]
		if h.Path != path || h.Addr.Start != w.start || h.Addr.End != w.end || h.Op != plan.OpReplace {
			t.Errorf("hunk %d: %s %d-%d %s, want %s %d-%d replace", i, h.Path, h.Addr.Start, h.Addr.End, h.Op, path, w.start, w.end)
		}
		if !reflect.DeepEqual(h.Body, w.body) {
			t.Errorf("hunk %d body %q, want %q", i, h.Body, w.body)
		}
		if h.Anchor != w.anchor {
			t.Errorf("hunk %d anchor %q, want %q (as the header splitter reads it)", i, h.Anchor, w.anchor)
		}
		if h.SHA != hex.EncodeToString(sum[:]) || !h.Raw {
			t.Errorf("hunk %d carries sha %q raw %v, want the file's sha and raw=true", i, h.SHA, h.Raw)
		}
	}
}

// ADR-161 Decision 3. A rewrite turns ast-grep's matches into whole-line
// replace hunks: matches on one line are one hunk, spans that touch are one
// hunk, each hunk is anchored on its first original line and pinned to the
// file's sha, and the read licenses each span and the line after it.
func TestARewritePlanGroupsSpansAndAnchorsThem(t *testing.T) {
	const a = "package a\n\nfunc f() {\n\tfoo(1, 2)\n\tx := foo(\n\t\t3,\n\t\t4)\n\t_ = foo(foo(5, 6), 7)\n\ts := \"naïve\"; foo(s, \"ŝ\")\n\t_ = x\n}\n"

	t.Run("single and multi-line matches, touching spans merged, a multibyte line", func(t *testing.T) {
		var r Rewrite
		hits := []RewriteHit{
			rwHit(t, a, "foo(1, 2)", "bar(2, 1)", 0),
			rwHit(t, a, "foo(\n\t\t3,\n\t\t4)", "bar(4, 3)", 0),
			rwHit(t, a, `foo(s, "ŝ")`, `bar("ŝ", s)`, 0),
		}
		r.File("a.go", []byte(a), hits)
		if len(r.Refused) > 0 {
			t.Fatalf("refused: %v", r.Refused)
		}
		// lines 4 and 5-7 touch: one hunk of four lines whose body is two
		rwCheck(t, &r, "a.go", a, []rwWant{
			{4, 7, []string{"\tbar(2, 1)", "\tx := bar(4, 3)"}, "\tfoo(1, 2)"},
			{9, 9, []string{"\ts := \"naïve\"; bar(\"ŝ\", s)"}, "\ts := \"naïve\"; foo(s, \"ŝ\")"},
		})
		// each span and the line after it, licensed and not shown
		o := r.Observed["a.go"]
		if !reflect.DeepEqual(o.Spans, [][2]int{{4, 8}, {9, 10}}) || !o.Written || o.Shown == nil || len(o.Shown) != 0 {
			t.Errorf("observation %+v, want spans 4-8 and 9-10, written, nothing shown", o)
		}
		if r.Files != 1 {
			t.Errorf("%d file(s), want 1", r.Files)
		}
	})

	t.Run("two matches on one line are one hunk; a gap keeps two hunks apart", func(t *testing.T) {
		d := "a\nfoo\nb\nfoo foo\nfoo\n"
		var r Rewrite
		r.File("g.txt", []byte(d), []RewriteHit{
			rwHit(t, d, "foo", "X", 0), rwHit(t, d, "foo", "Y", 1), rwHit(t, d, "foo", "Z", 2), rwHit(t, d, "foo", "W", 3),
		})
		rwCheck(t, &r, "g.txt", d, []rwWant{
			{2, 2, []string{"X"}, "foo"},
			{4, 5, []string{"Y Z", "W"}, "foo foo"},
		})
	})

	t.Run("an empty replacement is a real one", func(t *testing.T) {
		d := "x foo y\nfoo\nz\n"
		var r Rewrite
		r.File("e.txt", []byte(d), []RewriteHit{rwHit(t, d, "foo", "", 0), rwHit(t, d, "foo", "", 1)})
		rwCheck(t, &r, "e.txt", d, []rwWant{{1, 2, []string{"x  y", ""}, "x foo y"}})
	})

	t.Run("a rewrite that changes no line plans and licenses nothing", func(t *testing.T) {
		d := "foo\n"
		var r Rewrite
		r.File("n.txt", []byte(d), []RewriteHit{rwHit(t, d, "foo", "foo", 0)})
		if r.Sites != 0 || r.Plan.Len() != 0 || len(r.Observed["n.txt"].Spans) != 0 || r.Observed["n.txt"].Spans == nil {
			t.Errorf("sites %d plan %q observation %+v, want none planned and an empty, non-nil licence", r.Sites, r.Plan.String(), r.Observed["n.txt"])
		}
	})

	t.Run("a CRLF file's body carries no CR, and a replacement's own line break is a line", func(t *testing.T) {
		d := "a\r\nfoo(1)\r\nb\r\n"
		var r Rewrite
		r.File("c.txt", []byte(d), []RewriteHit{rwHit(t, d, "foo(1)", "bar(1,\nx)", 0)})
		rwCheck(t, &r, "c.txt", d, []rwWant{{2, 2, []string{"bar(1,", "x)"}, "foo(1)"}})
	})

	t.Run("an anchor and a path with quotes, a backslash, a tab and a multibyte rune round-trip through the header parser", func(t *testing.T) {
		path := `d/a b"c.go`
		line := "\tx := \"q\\\\u\" + 'z' // a=b ŝ foo"
		d := line + "\n"
		var r Rewrite
		r.File(path, []byte(d), []RewriteHit{rwHit(t, d, "foo", "bar", 0)})
		rwCheck(t, &r, path, d, []rwWant{{1, 1, []string{strings.Replace(line, "foo", "bar", 1)}, line}})
	})

	t.Run("a nested match is an overlap, refused by name, and plans nothing", func(t *testing.T) {
		// the offsets ast-grep 0.45.3 reported for foo(foo(5, 6), 7)
		var r Rewrite
		outer, inner := "foo(foo(5, 6), 7)", "foo(5, 6)"
		ro, ri := "bar(7, foo(5, 6))", "bar(6, 5)"
		s1, e1, s2, e2 := 59, 76, 63, 72
		r.File("a.go", []byte(a), []RewriteHit{
			{Text: &outer, Replacement: &ro, Start: &s1, End: &e1},
			{Text: &inner, Replacement: &ri, Start: &s2, End: &e2},
		})
		if len(r.Refused) != 1 || !strings.Contains(r.Refused[0], "overlap") || !strings.Contains(r.Refused[0], "a.go") {
			t.Errorf("refused %q, want one refusal naming a.go and the overlap", r.Refused)
		}
		if r.Plan.Len() != 0 || r.Sites != 0 {
			t.Errorf("a refused file planned %q", r.Plan.String())
		}
	})

	t.Run("a CRLF replacement's own CRLF is one break, and a CR inside a line is kept", func(t *testing.T) {
		d := "a\r\nfoo(1)\r\nb\r\n"
		var r Rewrite
		r.File("c.txt", []byte(d), []RewriteHit{rwHit(t, d, "foo(1)", "bar(1,\r\nx\ry)", 0)})
		rwCheck(t, &r, "c.txt", d, []rwWant{{2, 2, []string{"bar(1,", "x\ry)"}, "foo(1)"}})
	})

	refusals := []struct {
		name, data, want string
		hit              func(t *testing.T) RewriteHit
	}{
		{"a missing replacement", "foo\n", "replacement", func(t *testing.T) RewriteHit {
			h := rwHit(t, "foo\n", "foo", "x", 0)
			h.Replacement = nil
			return h
		}},
		{"a missing byteOffset", "foo\n", "byteOffset", func(t *testing.T) RewriteHit {
			h := rwHit(t, "foo\n", "foo", "x", 0)
			h.End = nil
			return h
		}},
		{"a range outside the file", "foo\n", "outside", func(t *testing.T) RewriteHit {
			h := rwHit(t, "foo\n", "foo", "x", 0)
			end := 99
			h.End, h.Text = &end, nil
			return h
		}},
		{"a file that is not what ast-grep read", "bar\n", "not what ast-grep read", func(t *testing.T) RewriteHit {
			return rwHit(t, "foo\n", "foo", "x", 0)
		}},
		{"a first line no anchor can name", "\nfoo\n", "anchor", func(t *testing.T) RewriteHit {
			// a match that begins in the line break of an empty first line
			s, e, txt, repl := 0, 4, "\nfoo", "x"
			return RewriteHit{Text: &txt, Replacement: &repl, Start: &s, End: &e}
		}},
		{"a CRLF replacement ending in a content CR", "a\r\nfoo(1)\r\nb\r\n", "CR", func(t *testing.T) RewriteHit {
			// the CR is the replacement's own: the consumed terminator is not in the text
			d := "a\r\nfoo(1)\r\nb\r\n"
			return rwHit(t, d, "foo(1)", "bar\r", 0)
		}},
	}
	for _, c := range refusals {
		t.Run(c.name+" is refused by name and plans nothing", func(t *testing.T) {
			var r Rewrite
			r.File("f.txt", []byte(c.data), []RewriteHit{c.hit(t)})
			if len(r.Refused) != 1 || !strings.Contains(r.Refused[0], c.want) || !strings.Contains(r.Refused[0], "f.txt") {
				t.Errorf("refused %q, want one refusal naming f.txt and %q", r.Refused, c.want)
			}
			if r.Plan.Len() != 0 || r.Sites != 0 {
				t.Errorf("a refused file planned %q", r.Plan.String())
			}
		})
	}
}
