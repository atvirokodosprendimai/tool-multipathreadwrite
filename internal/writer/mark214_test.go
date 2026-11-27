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

// ADR-214 Decision 6. The ledger follows the same rules after a write that
// puts a file's mark back: the record is the one the write kind earns, and a
// write that lands the same bytes leaves what a read showed. M is the UTF-8
// byte order mark, written as bytes.
func TestALedgerAfterAMarkedWriteFollowsTheSameRules(t *testing.T) {
	const (
		m    = "\xEF\xBB\xBF"
		orig = m + "one\ntwo\n"
	)
	file := func(t *testing.T, root, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, "m.txt"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	bytesOf := func(t *testing.T, root string) string {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(root, "m.txt"))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	record := func(t *testing.T, root string) seen.Observation {
		t.Helper()
		l, err := seen.Load(root)
		if err != nil {
			t.Fatal(err)
		}
		return l["m.txt"]
	}
	read := func(t *testing.T, root, body string, spans [][2]int) {
		t.Helper()
		if err := seen.Record(root, map[string]seen.Observation{"m.txt": {SHA: seen.SHA([]byte(body)), Spans: spans}}); err != nil {
			t.Fatal(err)
		}
	}
	// land runs the plan as the CLI does, reporting the result.
	land := func(t *testing.T, root string, in []apply.Input, opts apply.Options) apply.Result {
		t.Helper()
		p, err := Prepare(Request{Root: root, In: in, Check: CheckOff, Opts: opts})
		if err != nil {
			t.Fatal(err)
		}
		l, err := p.Land()
		if err != nil {
			t.Fatal(err)
		}
		return l.Res
	}
	native := func(anchor string, body string) []apply.Input {
		return []apply.Input{{Path: "m.txt", Op: "replace", Start: 1, End: 1, Anchor: anchor, Lines: -1, Body: []string{body}}}
	}
	quoted := func(t *testing.T, root, replace string) []apply.Input {
		t.Helper()
		out, err := ingest.CompileSearchReplace(root, []byte("m.txt\n<<<<<<< SEARCH\n"+m+"one\n=======\n"+replace+"\n>>>>>>> REPLACE\n"))
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
				Occurrence: h.Occurrence, CountedBody: h.CountedBody,
				Body: h.Body, SHA: h.SHA, Lines: h.Lines, Anchor: h.Anchor, Quote: h.Quote, SrcLine: h.SrcLine,
			})
		}
		return in
	}

	t.Run("a_native_replace_of_line_1_is_whole_and_shows_nothing", func(t *testing.T) {
		root := checkout(t)
		file(t, root, orig)
		read(t, root, orig, nil)
		if res := land(t, root, native("", "ONE"), apply.Options{}); !res.Applied || res.Failed != 0 {
			t.Fatalf("first write: %+v", res.Hunks)
		}
		if got := bytesOf(t, root); got != m+"ONE\ntwo\n" {
			t.Fatalf("file = % x", got)
		}
		if r := record(t, root); !r.Written || !r.Whole() || len(r.Shown) != 0 || r.ServedLine(1) {
			t.Errorf("record %+v, want whole, written and nothing shown", r)
		}
		if res := land(t, root, native("", "TWO"), apply.Options{}); res.Applied || res.Failed == 0 {
			t.Errorf("a bare 1 replace on a line nothing showed was not refused: %+v", res.Hunks)
		}
		if got := bytesOf(t, root); got != m+"ONE\ntwo\n" {
			t.Errorf("a refusal changed the file: % x", got)
		}
		if res := land(t, root, native("ONE", "TWO"), apply.Options{}); !res.Applied || res.Failed != 0 {
			t.Errorf("anchor ONE did not land: %+v", res.Hunks)
		}
	})
	t.Run("a_quote_only_write_licenses_line_1", func(t *testing.T) {
		root := checkout(t)
		file(t, root, orig)
		if res := land(t, root, quoted(t, root, "ONE"), apply.Options{Computed: true}); !res.Applied || res.Failed != 0 {
			t.Fatalf("quoted write: %+v", res.Hunks)
		}
		if got := bytesOf(t, root); got != m+"ONE\ntwo\n" {
			t.Fatalf("file = % x", got)
		}
		if r := record(t, root); !r.Written || r.Whole() || !reflect.DeepEqual(r.Spans, [][2]int{{1, 1}}) || len(r.Shown) != 0 {
			t.Errorf("record %+v, want written, licensed 1-1, nothing shown", r)
		}
		if res := land(t, root, native("", "TWO"), apply.Options{}); !res.Applied || res.Failed != 0 {
			t.Errorf("a bare 1 replace on the licensed line did not land: %+v", res.Hunks)
		}
	})
	t.Run("a_quote_that_lands_the_same_bytes_writes_them_again_and_keeps_a_showing", func(t *testing.T) {
		root := checkout(t)
		file(t, root, orig)
		read(t, root, orig, [][2]int{{1, 1}})
		before := record(t, root)
		if !before.ServedLine(1) {
			t.Fatalf("line 1 was not shown by its read: %+v", before)
		}
		res := land(t, root, quoted(t, root, "one"), apply.Options{Computed: true})
		if res.Failed != 0 {
			t.Fatalf("SEARCH Mone REPLACE one: %+v", res.Hunks)
		}
		if got := bytesOf(t, root); got != orig {
			t.Errorf("file = % x, want the same bytes", got)
		}
		// The same bytes are written again, as for any file (Zy, 2026-11-27):
		// Written is the write's; the showing is what must survive.
		if after := record(t, root); after.SHA != before.SHA || !reflect.DeepEqual(after.Spans, before.Spans) || !after.ServedLine(1) {
			t.Errorf("record %+v after, %+v before: a showing must survive a write of the same bytes", after, before)
		}
	})
	t.Run("a_dry_run_leaves_the_ledger_alone", func(t *testing.T) {
		root := checkout(t)
		file(t, root, orig)
		read(t, root, orig, [][2]int{{1, 1}})
		before := record(t, root)
		if res := land(t, root, native("", "ONE"), apply.Options{DryRun: true}); res.Failed != 0 {
			t.Fatalf("dry run: %+v", res.Hunks)
		}
		if got := bytesOf(t, root); got != orig {
			t.Errorf("a dry run changed the file: % x", got)
		}
		if after := record(t, root); !reflect.DeepEqual(after, before) {
			t.Errorf("a dry run moved the record: %+v -> %+v", before, after)
		}
	})
}
