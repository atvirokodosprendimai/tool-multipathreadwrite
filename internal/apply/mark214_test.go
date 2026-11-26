package apply

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/ingest"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
)

// ADR-214 T1. M is the UTF-8 byte order mark, written as bytes so no source
// line holds the invisible character.
const m214 = "\xEF\xBB\xBF"

func hex214(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// in214 numbers a plan's hunks in order, as the parser does.
func in214(hs ...Input) []Input {
	for i := range hs {
		hs[i].Index, hs[i].Lines = i, -1
		if hs[i].Path == "" {
			hs[i].Path = "m.txt"
		}
	}
	return hs
}

func rep214(a, b int, anchor string, body ...string) Input {
	return Input{Op: "replace", Start: a, End: b, Anchor: anchor, Body: body}
}
func del214(a, b int) Input { return Input{Op: "delete", Start: a, End: b} }
func insB214(n int, body ...string) Input {
	return Input{Op: "insert-before", Start: n, End: n, Body: body}
}
func insA214(n int, body ...string) Input {
	return Input{Op: "insert-after", Start: n, End: n, Body: body}
}

// whole214 licenses every line of each named file, as a whole-file read does.
func whole214(t *testing.T, root string, names ...string) Options {
	t.Helper()
	l := map[string]Seen{}
	for _, n := range names {
		l[n] = Seen{SHA: shaOfFile(t, root, n)}
	}
	return Options{Seen: l}
}

// planOf214 compiles a foreign document to the engine's input as the CLI does.
func planOf214(root string, compile func(string, []byte) ([]byte, error), doc string) ([]Input, error) {
	out, err := compile(root, []byte(doc))
	if err != nil {
		return nil, err
	}
	hunks, err := plan.Parse(strings.NewReader(string(out)))
	if err != nil {
		return nil, err
	}
	var in []Input
	for _, h := range hunks {
		in = append(in, Input{
			Path: h.Path, Start: h.Addr.Start, End: h.Addr.End, Op: string(h.Op),
			StartPat: h.Addr.StartPat, EndPat: h.Addr.EndPat, RelEnd: h.Addr.RelEnd, Occurrence: h.Occurrence, CountedBody: h.CountedBody,
			Body: h.Body, SHA: h.SHA, Lines: h.Lines, Anchor: h.Anchor, Quote: h.Quote, SrcLine: h.SrcLine,
		})
	}
	return in, nil
}

func TestAWriteKeepsTheFilesLeadingMarkAtByteZero(t *testing.T) {
	const sr = "m.txt\n<<<<<<< SEARCH\n%s\n=======\n%s\n>>>>>>> REPLACE\n"
	sr214 := func(old, neu string) func(string) ([]Input, error) {
		return func(root string) ([]Input, error) {
			return planOf214(root, ingest.CompileSearchReplace, strings.Replace(strings.Replace(sr, "%s", old, 1), "%s", neu, 1))
		}
	}
	ap214 := func(old, neu string) func(string) ([]Input, error) {
		return func(root string) ([]Input, error) {
			return planOf214(root, ingest.CompileApplyPatch, "*** Begin Patch\n*** Update File: m.txt\n@@\n-"+old+"\n+"+neu+"\n*** End Patch\n")
		}
	}
	type row struct {
		name    string
		orig    string
		in      []Input
		foreign func(root string) ([]Input, error)
		want    string // the bytes that land
		refuse  bool   // the plan is refused and the bytes stay
	}
	const mm = m214 + m214
	rows := []row{
		{name: "replace_1_without_the_mark", orig: m214 + "one\ntwo\n", in: in214(rep214(1, 1, "", "ONE")), want: m214 + "ONE\ntwo\n"},
		{name: "replace_1_with_the_mark", orig: m214 + "one\ntwo\n", in: in214(rep214(1, 1, "", m214+"ONE")), want: m214 + "ONE\ntwo\n"},
		{name: "delete_1", orig: m214 + "one\ntwo\n", in: in214(del214(1, 1)), want: m214 + "two\n"},
		{name: "insert_before_1", orig: m214 + "one\ntwo\n", in: in214(insB214(1, "zero")), want: m214 + "zero\none\ntwo\n"},
		{name: "insert_after_0", orig: m214 + "one\ntwo\n", in: in214(insA214(0, "zero")), want: m214 + "zero\none\ntwo\n"},
		{name: "replace_1_2", orig: m214 + "one\ntwo\n", in: in214(rep214(1, 2, "one", "A", "B")), want: m214 + "A\nB\n"},
		{name: "delete_1_2", orig: m214 + "one\ntwo\n", in: in214(del214(1, 2)), want: m214},
		{name: "replace_1_and_2_in_one_plan", orig: m214 + "one\ntwo\n", in: in214(rep214(1, 1, "", "ONE"), rep214(2, 2, "", "TWO")), want: m214 + "ONE\nTWO\n"},
		{name: "replace_1_with_an_empty_line", orig: m214 + "one\ntwo\n", in: in214(rep214(1, 1, "", "")), want: m214 + "\ntwo\n"},
		{name: "insert_and_replace_at_one_position", orig: m214 + "one\ntwo\n", in: in214(insB214(1, "zero"), rep214(1, 1, "", "ONE")), want: m214 + "zero\nONE\ntwo\n"},
		{name: "replace_2_and_insert_after_1", orig: m214 + "one\ntwo\n", in: in214(rep214(2, 2, "", "TWO"), insA214(1, "x")), want: m214 + "one\nx\nTWO\n"},
		{name: "two_marks_replace_1", orig: mm + "one\n", in: in214(rep214(1, 1, "", "X")), want: m214 + "X\n"},
		{name: "two_marks_replace_2", orig: mm + "one\ntwo\n", in: in214(rep214(2, 2, "", "TWO")), want: mm + "one\nTWO\n"},
		{name: "promoted_line_keeps_its_own_mark", orig: m214 + "one\n" + m214 + "two\n", in: in214(del214(1, 1)), want: mm + "two\n"},
		{name: "only_the_mark_insert_after_0", orig: m214, in: in214(insA214(0, "zero")), want: m214 + "zero\n"},
		{name: "only_the_mark_insert_before_1", orig: m214, in: in214(insB214(1, "zero")), want: m214 + "zero\n"},
		{name: "only_the_mark_delete", orig: m214, in: in214(del214(1, 1)), want: m214},
		{name: "only_the_mark_replace", orig: m214, in: in214(rep214(1, 1, "", "x")), want: m214 + "x"},
		{name: "raw_rewrite_of_the_signature_only", orig: mm + "one\n", in: in214(rep214(1, 1, "", m214+"one")), want: m214 + "one\n"},
		{name: "raw_rewrite_of_the_whole_line", orig: mm + "one\n", in: in214(rep214(1, 1, "", "ONE")), want: m214 + "ONE\n"},
		{name: "sub_dot_one", orig: m214 + "one\ntwo\n", in: in214(rep214(1, 1, "", "ONE")), want: m214 + "ONE\ntwo\n"},
		{name: "search_one_is_refused", orig: m214 + "one\ntwo\n", foreign: sr214("one", "ONE"), refuse: true},
		{name: "search_with_the_mark", orig: m214 + "one\ntwo\n", foreign: sr214(m214+"one", "ONE"), want: m214 + "ONE\ntwo\n"},
		{name: "search_with_the_mark_replace_with_it", orig: m214 + "one\ntwo\n", foreign: sr214(m214+"one", m214+"ONE"), want: m214 + "ONE\ntwo\n"},
		{name: "patch_one_is_refused", orig: m214 + "one\ntwo\n", foreign: ap214("one", "ONE"), refuse: true},
		{name: "patch_with_the_mark", orig: m214 + "one\ntwo\n", foreign: ap214(m214+"one", "ONE"), want: m214 + "ONE\ntwo\n"},
		{name: "patch_with_the_mark_replace_with_it", orig: m214 + "one\ntwo\n", foreign: ap214(m214+"one", m214+"ONE"), want: m214 + "ONE\ntwo\n"},
		{name: "crlf", orig: m214 + "one\r\ntwo\r\n", in: in214(rep214(1, 1, "", "ONE")), want: m214 + "ONE\r\ntwo\r\n"},
		{name: "cr_only", orig: m214 + "one\rtwo\r", in: in214(rep214(1, 1, "", "ONE")), want: m214 + "ONE\rtwo\r"},
		{name: "an_unmarked_file_gains_no_mark", orig: "one\ntwo\n", in: in214(rep214(1, 1, "", "ONE")), want: "ONE\ntwo\n"},
		{name: "an_unmarked_file_takes_a_marked_body", orig: "one\ntwo\n", in: in214(rep214(1, 1, "", m214+"ONE")), want: m214 + "ONE\ntwo\n"},
	}
	ran := 0
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			ran++
			root := t.TempDir()
			write(t, root, "m.txt", r.orig)
			in := r.in
			opt := whole214(t, root, "m.txt")
			if r.foreign != nil {
				opt.Computed = true
				var err error
				in, err = r.foreign(root)
				if r.refuse {
					if err == nil {
						t.Fatal("the foreign plan compiled; want a refusal")
					}
					if got := read(t, root, "m.txt"); got != r.orig {
						t.Errorf("bytes changed on a refusal: %q", got)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			res, err := Apply(root, in, opt)
			if err != nil || res.Failed != 0 {
				t.Fatalf("Apply: %v failed=%d %+v", err, res.Failed, res.Hunks)
			}
			got := read(t, root, "m.txt")
			if got != r.want {
				t.Errorf("file = % x, want % x", got, r.want)
			}
			if res.Files[0].SHABefore != hex214(r.orig) {
				t.Errorf("sha_before = %s, want the raw %s", res.Files[0].SHABefore, hex214(r.orig))
			}
			if res.Files[0].SHAAfter != hex214(got) {
				t.Errorf("sha_after = %s, want the landed %s", res.Files[0].SHAAfter, hex214(got))
			}
		})
	}
	if ran < len(rows) {
		t.Fatalf("only %d of %d rows ran", ran, len(rows))
	}

	t.Run("rename_keeps_every_byte", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, "m.txt", m214+"one\ntwo\n")
		res, err := Apply(root, in214(Input{Op: "rename", Body: []string{"g.txt"}}), whole214(t, root, "m.txt"))
		if err != nil || res.Failed != 0 {
			t.Fatalf("Apply: %v %+v", err, res.Hunks)
		}
		if got := read(t, root, "g.txt"); got != m214+"one\ntwo\n" {
			t.Errorf("g.txt = % x", got)
		}
		if _, err := os.Lstat(filepath.Join(root, "m.txt")); err == nil {
			t.Error("m.txt is still there")
		}
		if n, dst := recordsFor(res, "g.txt"); n != 1 || dst.SHAAfter != hex214(m214+"one\ntwo\n") {
			t.Errorf("g.txt record: %d %+v", n, dst)
		}
	})
	t.Run("rename_with_a_line_1_edit", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, "m.txt", m214+"one\ntwo\n")
		res, err := Apply(root, in214(rep214(1, 1, "", "ONE"), Input{Op: "rename", Body: []string{"g.txt"}}), whole214(t, root, "m.txt"))
		if err != nil || res.Failed != 0 {
			t.Fatalf("Apply: %v %+v", err, res.Hunks)
		}
		if got := read(t, root, "g.txt"); got != m214+"ONE\ntwo\n" {
			t.Errorf("g.txt = % x", got)
		}
	})
	t.Run("unlink_is_not_a_line_edit", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, "m.txt", m214+"one\n")
		res, err := Apply(root, in214(Input{Op: "unlink"}), whole214(t, root, "m.txt"))
		if err != nil || res.Failed != 0 {
			t.Fatalf("Apply: %v %+v", err, res.Hunks)
		}
		if _, err := os.Lstat(filepath.Join(root, "m.txt")); err == nil {
			t.Error("m.txt is still there")
		}
	})
	t.Run("create_of_a_missing_file_keeps_its_body_mark", func(t *testing.T) {
		root := t.TempDir()
		res, err := Apply(root, in214(Input{Op: "create", Body: []string{m214 + "hello"}}), Options{})
		if err != nil || res.Failed != 0 {
			t.Fatalf("Apply: %v %+v", err, res.Hunks)
		}
		if got := read(t, root, "m.txt"); got != m214+"hello\n" {
			t.Errorf("m.txt = % x", got)
		}
	})
}

func TestAWriteReceiptDescribesTheLandedFile(t *testing.T) {
	const orig = m214 + "one\ntwo\n"
	for _, dry := range []bool{true, false} {
		root := t.TempDir()
		write(t, root, "m.txt", orig)
		opt := whole214(t, root, "m.txt")
		opt.DryRun = dry
		res, err := Apply(root, in214(del214(1, 2)), opt)
		if err != nil || res.Failed != 0 {
			t.Fatalf("dry=%v Apply: %v %+v", dry, err, res.Hunks)
		}
		f := res.Files[0]
		if f.LinesTo != 1 || f.SHAAfter != hex214(m214) {
			t.Errorf("dry=%v: lines_after %d sha_after %s, want 1 and the sha of the mark", dry, f.LinesTo, f.SHAAfter)
		}
		if h := res.Hunks[0]; h.Removed != 2 || h.Added != 0 {
			t.Errorf("dry=%v: the hunk counts -%d +%d, want -2 +0", dry, h.Removed, h.Added)
		}
		if !dry {
			if got := read(t, root, "m.txt"); got != m214 {
				t.Errorf("m.txt = % x, want the mark alone", got)
			}
		}
	}
	t.Run("delete_all_plus_rename", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, "m.txt", orig)
		res, err := Apply(root, in214(del214(1, 2), Input{Op: "rename", Body: []string{"g.txt"}}), whole214(t, root, "m.txt"))
		if err != nil || res.Failed != 0 {
			t.Fatalf("Apply: %v %+v", err, res.Hunks)
		}
		if n, dst := recordsFor(res, "g.txt"); n != 1 || dst.LinesTo != 1 || dst.SHAAfter != hex214(m214) {
			t.Errorf("g.txt record: %d %+v, want 1 line and the sha of the mark", n, dst)
		}
	})
	t.Run("insert_into_the_mark_alone_counts_the_landed_line", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, "m.txt", m214)
		res, err := Apply(root, in214(insB214(1, "zero")), whole214(t, root, "m.txt"))
		if err != nil || res.Failed != 0 {
			t.Fatalf("Apply: %v %+v", err, res.Hunks)
		}
		if f := res.Files[0]; f.LinesTo != 1 {
			t.Errorf("lines_after = %d, want 1: the landed Mzero is one line", f.LinesTo)
		}
	})
	t.Run("a_sub_preview_shows_the_landed_line", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, "m.txt", orig)
		opt := whole214(t, root, "m.txt")
		opt.DryRun, opt.Preview = true, true
		res, err := Apply(root, in214(rep214(1, 1, "", "ONE")), opt)
		if err != nil || res.Failed != 0 {
			t.Fatalf("Apply: %v %+v", err, res.Hunks)
		}
		if a := res.Hunks[0].After; len(a) != 1 || a[0] != m214+"ONE" {
			t.Errorf("preview after = %q, want the mark and ONE", a)
		}
	})
	t.Run("the_pad_shows_the_adjusted_tail", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, "m.txt", orig)
		opt := whole214(t, root, "m.txt")
		opt.EchoPad = 1
		res, err := Apply(root, in214(insB214(1, "zero")), opt)
		if err != nil || res.Failed != 0 {
			t.Fatalf("Apply: %v %+v", err, res.Hunks)
		}
		if e := res.Hunks[0].Echo; len(e) != 1 || e[0] != "    2| one" {
			t.Errorf("echo = %q, want line 2 as one with no mark", e)
		}
	})
	t.Run("the_closer_hint_is_judged_on_the_submitted_body", func(t *testing.T) {
		closer := func(orig string) string {
			root := t.TempDir()
			write(t, root, "m.go", orig)
			in := in214(rep214(1, 1, "", "}"))
			in[0].Path = "m.go"
			res, err := Apply(root, in, whole214(t, root, "m.go"))
			if err != nil || res.Failed != 0 {
				t.Fatalf("Apply: %v %+v", err, res.Hunks)
			}
			return res.Hunks[0].Closer
		}
		plain, marked := closer("x\n}\n"), closer(m214+"x\n}\n")
		if plain == "" || marked != plain {
			t.Errorf("closer on a marked file %q, on an unmarked one %q: want the same, non-empty", marked, plain)
		}
	})
}

func TestAFromCopyOfALineOneCarriesNoMark(t *testing.T) {
	type row struct {
		name, src, dst string
		at             int
		op             string
		want           string
		first          string
	}
	rows := []row{
		{"into_the_middle_of_an_unmarked_file", m214 + "one\n", "a\nb\n", 1, "insert-after", "a\none\nb\n", "one"},
		{"to_the_top_of_a_marked_file", m214 + "one\n", m214 + "a\nb\n", 1, "insert-before", m214 + "one\na\nb\n", "one"},
		{"to_the_top_of_an_unmarked_file", m214 + "one\n", "a\nb\n", 1, "insert-before", "one\na\nb\n", "one"},
		{"a_double_mark_source_to_a_marked_top", m214 + m214 + "one\n", m214 + "a\nb\n", 1, "insert-before", m214 + "one\na\nb\n", m214 + "one"},
		{"a_source_that_is_only_the_mark", m214, "a\nb\n", 1, "insert-after", "a\n\nb\n", ""},
		{"an_unmarked_source_as_before", "one\n", "a\nb\n", 1, "insert-after", "a\none\nb\n", "one"},
	}
	ran := 0
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			ran++
			root := t.TempDir()
			write(t, root, "src.txt", r.src)
			write(t, root, "dst.txt", r.dst)
			in := []Input{{Path: "dst.txt", Op: r.op, Start: r.at, End: r.at, Lines: -1, From: "src.txt", FromStart: 1, FromEnd: 1}}
			if r.op == "insert-before" {
				in[0].Start, in[0].End = 1, 1
			}
			res, err := Apply(root, in, whole214(t, root, "src.txt", "dst.txt"))
			if err != nil || res.Failed != 0 {
				t.Fatalf("Apply: %v failed=%d %+v", err, res.Failed, res.Hunks)
			}
			if got := read(t, root, "dst.txt"); got != r.want {
				t.Errorf("dst = % x, want % x", got, r.want)
			}
			h := res.Hunks[0]
			if h.CopiedFirst != r.first || h.CopiedLast != r.first {
				t.Errorf("copied_first %q copied_last %q, want %q", h.CopiedFirst, h.CopiedLast, r.first)
			}
		})
	}
	if ran < len(rows) {
		t.Fatalf("only %d of %d rows ran", ran, len(rows))
	}
	t.Run("the_source_sha_gate_stays_raw", func(t *testing.T) {
		root := t.TempDir()
		write(t, root, "src.txt", m214+"one\n")
		write(t, root, "dst.txt", "a\n")
		opt := Options{Seen: map[string]Seen{
			"src.txt": {SHA: hex214("one\n")},
			"dst.txt": {SHA: shaOfFile(t, root, "dst.txt")},
		}}
		res, err := Apply(root, []Input{{Path: "dst.txt", Op: "insert-after", Start: 1, End: 1, Lines: -1, From: "src.txt", FromStart: 1, FromEnd: 1}}, opt)
		if err != nil || res.Failed == 0 {
			t.Errorf("a ledger sha of the stripped text licensed the copy: %v failed=%d", err, res.Failed)
		}
	})
}

func TestAMarkedFileRefusalTeachesTheMark(t *testing.T) {
	const taught = "line 1 begins with a UTF-8 byte order mark (U+FEFF); an exact match must include it"
	root := t.TempDir()
	write(t, root, "m.txt", m214+"one\ntwo\n")
	res, err := Apply(root, in214(Input{Op: "delete", Start: 1, End: 1, Body: []string{"one"}}), whole214(t, root, "m.txt"))
	if err != nil || res.Failed == 0 {
		t.Fatalf("the delete expecting the visible text was not refused: %v %+v", err, res.Hunks)
	}
	if r := res.Hunks[0].Reason; !strings.Contains(r, taught) || !strings.Contains(r, `^\x{FEFF}one$`) {
		t.Errorf("delete refusal %q does not teach the mark and the retry", r)
	}
	for name, compile := range map[string]func() error{
		"search": func() error {
			_, e := ingest.CompileSearchReplace(root, []byte("m.txt\n<<<<<<< SEARCH\none\n=======\nONE\n>>>>>>> REPLACE\n"))
			return e
		},
		"patch": func() error {
			_, e := ingest.CompileApplyPatch(root, []byte("*** Begin Patch\n*** Update File: m.txt\n@@\n-one\n+ONE\n*** End Patch\n"))
			return e
		},
	} {
		if e := compile(); e == nil || !strings.Contains(e.Error(), taught) {
			t.Errorf("%s miss %v does not teach the mark", name, e)
		}
	}
	// A SEARCH one still matches line 2 of Mone/one: the guide says so.
	write(t, root, "n.txt", m214+"one\none\n")
	if _, e := ingest.CompileSearchReplace(root, []byte("n.txt\n<<<<<<< SEARCH\none\n=======\nONE\n>>>>>>> REPLACE\n")); e != nil {
		t.Errorf("SEARCH one on Mone/one: %v", e)
	}
	// No refusal on an unmarked file mentions a mark.
	write(t, root, "u.txt", "one\ntwo\n")
	res, err = Apply(root, in214(Input{Path: "u.txt", Op: "delete", Start: 1, End: 1, Body: []string{"nope"}}), whole214(t, root, "u.txt"))
	if err != nil || res.Failed == 0 || strings.Contains(res.Hunks[0].Reason, "byte order mark") {
		t.Errorf("unmarked refusal: %v %+v", err, res.Hunks)
	}
	if _, e := ingest.CompileSearchReplace(root, []byte("u.txt\n<<<<<<< SEARCH\nnope\n=======\nONE\n>>>>>>> REPLACE\n")); e == nil || strings.Contains(e.Error(), "byte order mark") {
		t.Errorf("unmarked SEARCH miss: %v", e)
	}
}
