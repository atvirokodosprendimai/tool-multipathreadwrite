package writer

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/ingest"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-208 Decision 1a. A write whose every hunk on a file was a verified
// search_replace quote records the file licensed and not shown: Written, the lines
// it wrote as the licence, nothing shown. A read-shaped record let the lines the
// caller wrote and the lines a read showed look alike, which hid the closer a
// later multi-line replace leaves behind.
func TestASearchReplaceOnlyWriteIsRecordedLicensedNotShown(t *testing.T) {
	const (
		five = "package a\n\nfunc A() {}\n\nvar z = 1\n"
		doc  = "f.go\n<<<<<<< SEARCH\nvar z = 1\n=======\nvar z = 2\nfunc B() {\n\tif y {\n\t}\n}\n>>>>>>> REPLACE\n"
	)
	quoted := func(t *testing.T, root string) []apply.Input {
		t.Helper()
		out, err := ingest.CompileSearchReplace(root, []byte(doc))
		if err != nil {
			t.Fatal(err)
		}
		hunks, err := plan.Parse(strings.NewReader(string(out)))
		if err != nil {
			t.Fatalf("%v\n%s", err, out)
		}
		var in []apply.Input
		for _, h := range hunks {
			in = append(in, apply.Input{
				Path: h.Path, Start: h.Addr.Start, End: h.Addr.End, Op: string(h.Op),
				StartPat: h.Addr.StartPat, EndPat: h.Addr.EndPat, RelEnd: h.Addr.RelEnd, Occurrence: h.Occurrence, CountedBody: h.CountedBody,
				Body: h.Body, SHA: h.SHA, Lines: h.Lines, Anchor: h.Anchor, Quote: h.Quote, SrcLine: h.SrcLine, Index: h.Index,
			})
		}
		return in
	}
	land := func(t *testing.T, root string, in []apply.Input) {
		t.Helper()
		p, err := Prepare(Request{Root: root, In: in, Check: CheckOff, Opts: apply.Options{Computed: true}})
		if err != nil {
			t.Fatal(err)
		}
		l, err := p.Land()
		if err != nil || !l.Res.Applied || l.Res.Failed != 0 || l.LedgerErr != nil {
			t.Fatalf("land: err %v, applied %v, failed %d, ledger %v: %+v", err, l.Res.Applied, l.Res.Failed, l.LedgerErr, l.Res.Hunks)
		}
	}
	file := func(t *testing.T, root, name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	record := func(t *testing.T, root, name string) seen.Observation {
		t.Helper()
		l, err := seen.Load(root)
		if err != nil {
			t.Fatal(err)
		}
		return l[name]
	}
	readWhole := func(t *testing.T, root, name, body string) {
		t.Helper()
		if err := seen.Record(root, map[string]seen.Observation{name: {SHA: seen.SHA([]byte(body))}}); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("quote_only", func(t *testing.T) {
		root := checkout(t)
		file(t, root, "f.go", five)
		land(t, root, quoted(t, root))
		got := record(t, root, "f.go")
		if !got.Written || got.Whole() || !reflect.DeepEqual(got.Spans, [][2]int{{5, 9}}) || len(got.Shown) != 0 {
			t.Errorf("the record is %+v, want written, licensed 5-9 and nothing shown", got)
		}
		if got.ServedLine(9) {
			t.Error("a line the write produced counts as shown")
		}
	})
	t.Run("mixed", func(t *testing.T) {
		root := checkout(t)
		file(t, root, "f.go", five)
		readWhole(t, root, "f.go", five)
		in := append(quoted(t, root), apply.Input{Path: "f.go", Op: "replace", Start: 1, End: 1, Anchor: "package a", Lines: -1, Body: []string{"package a // x"}, SrcLine: 9, Index: 9})
		land(t, root, in)
		if got := record(t, root, "f.go"); !got.Written || !got.Whole() {
			t.Errorf("a write with a native hunk on the file is %+v, want whole and written", got)
		}
	})
	t.Run("two_files", func(t *testing.T) {
		root := checkout(t)
		file(t, root, "f.go", five)
		file(t, root, "g.go", "package g\nfunc G() {}\n")
		readWhole(t, root, "g.go", "package g\nfunc G() {}\n")
		in := append(quoted(t, root), apply.Input{Path: "g.go", Op: "replace", Start: 2, End: 2, Anchor: "func G", Lines: -1, Body: []string{"func G() { _ = 1 }"}, SrcLine: 9, Index: 9})
		land(t, root, in)
		f, g := record(t, root, "f.go"), record(t, root, "g.go")
		if !f.Written || f.Whole() || !reflect.DeepEqual(f.Spans, [][2]int{{5, 9}}) || len(f.Shown) != 0 {
			t.Errorf("f.go (quoted only) is %+v", f)
		}
		if !g.Written || !g.Whole() {
			t.Errorf("g.go (edited natively) is %+v, want whole and written", g)
		}
	})
	// A second quoted write that changes one line at a new sha leaves a record of that line alone, and
	// nothing shown: a showing of line 9 at the first write's sha does not carry over to the new file.
	t.Run("later_one_line_write", func(t *testing.T) {
		root := checkout(t)
		file(t, root, "f.go", five)
		land(t, root, quoted(t, root))
		first := record(t, root, "f.go")
		if err := seen.Record(root, map[string]seen.Observation{"f.go": {SHA: first.SHA, Spans: [][2]int{{9, 9}}}}); err != nil {
			t.Fatal(err)
		}
		if got := record(t, root, "f.go"); !got.ServedLine(9) {
			t.Fatalf("line 9 is not shown after a read of it: %+v", got)
		}
		out, err := ingest.CompileSearchReplace(root, []byte("f.go\n<<<<<<< SEARCH\nvar z = 2\n=======\nvar z = 3\n>>>>>>> REPLACE\n"))
		if err != nil {
			t.Fatal(err)
		}
		hunks, err := plan.Parse(strings.NewReader(string(out)))
		if err != nil {
			t.Fatalf("%v\n%s", err, out)
		}
		var in []apply.Input
		for _, h := range hunks {
			in = append(in, apply.Input{
				Path: h.Path, Start: h.Addr.Start, End: h.Addr.End, Op: string(h.Op), Occurrence: h.Occurrence, CountedBody: h.CountedBody,
				Body: h.Body, SHA: h.SHA, Lines: h.Lines, Anchor: h.Anchor, Quote: h.Quote, SrcLine: h.SrcLine, Index: h.Index,
			})
		}
		land(t, root, in)
		got := record(t, root, "f.go")
		if got.SHA == first.SHA || !got.Written || got.Whole() || !reflect.DeepEqual(got.Spans, [][2]int{{5, 5}}) || len(got.Shown) != 0 {
			t.Errorf("the second write left %+v, want written, licensed 5-5 and nothing shown at a new sha", got)
		}
	})
}
