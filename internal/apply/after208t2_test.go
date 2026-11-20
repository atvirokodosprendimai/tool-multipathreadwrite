package apply

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/ingest"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-208 Decision 1a. A written-shaped record, whole or partial, meets a multi-line
// replace's line after only by a showing at the current sha. f.go is the record's
// repro after its search_replace write: nine lines, 5-9 written by the caller's
// earlier call (licensed, not shown), and the plan replaces 6-8 with a body ending
// in a closer, leaving line 9's closer behind.
func TestAMultiLineReplaceWhoseLineAfterASearchReplaceWroteIsRefusedUntilShown(t *testing.T) {
	const (
		body  = "package a\n\nfunc A() {}\n\nvar z = 2\nfunc B() {\n\tif y {\n\t}\n}\n"
		twice = "package a\n\nfunc A() {}\n\nvar z = 2\n\t}\nfunc B() {\n\t}\n}\n" // `\t}` on lines 6 and 8
		old   = "package a\n\nfunc A() {}\n\nvar z = 1\n"
	)
	sha, olderSHA, twiceSHA := seen.SHA([]byte(body)), seen.SHA([]byte(old)), seen.SHA([]byte(twice))
	rep := func(start, end int) []Input {
		return []Input{{Path: "f.go", Op: "replace", Start: start, End: end, Anchor: "func B", Lines: -1, Body: []string{"func B() {", "\tz()", "}"}}}
	}
	planOf := func(t *testing.T, root, doc string) []Input {
		t.Helper()
		out, err := ingest.CompileSearchReplace(root, []byte(doc))
		if err != nil {
			t.Fatal(err)
		}
		hunks, err := plan.Parse(strings.NewReader(string(out)))
		if err != nil {
			t.Fatalf("%v\n%s", err, out)
		}
		var in []Input
		for _, h := range hunks {
			in = append(in, Input{
				Path: h.Path, Start: h.Addr.Start, End: h.Addr.End, Op: string(h.Op),
				StartPat: h.Addr.StartPat, EndPat: h.Addr.EndPat, RelEnd: h.Addr.RelEnd, Occurrence: h.Occurrence, CountedBody: h.CountedBody,
				Body: h.Body, SHA: h.SHA, Lines: h.Lines, Anchor: h.Anchor, Quote: h.Quote, SrcLine: h.SrcLine, Index: h.Index,
			})
		}
		return in
	}
	noTail := "f.go\n<<<<<<< SEARCH\nfunc B() {\n\tif y {\n\t}\n=======\nfunc B() {\n\tz()\n}\n>>>>>>> REPLACE\n"
	keptTail := "f.go\n<<<<<<< SEARCH\nfunc B() {\n\tif y {\n\t}\n}\n=======\nfunc B() {\n\tz()\n}\n>>>>>>> REPLACE\n"
	w := func(shown [][2]int, spans ...[2]int) Seen {
		return Seen{SHA: sha, Written: true, Spans: spans, Shown: shown}
	}
	w59 := w(nil, [2]int{5, 9})
	readOf := func(spans ...[2]int) Seen { return Seen{SHA: sha, Spans: spans} }
	const ref = "needs a served line after 8"
	const remedy = "mrw read --max-cols 0 f.go:6-9"

	cases := []struct {
		name   string
		file   string
		ledger map[string]Seen
		in     func(t *testing.T, root string) []Input
		opt    Options
		want   string // "ok", or what the refusal's reason must carry
		suffix string // what a written-shaped refusal ends with
		alias  bool   // g.go is a link to f.go and the plan names g.go
	}{
		{name: "written_line_after", ledger: map[string]Seen{"f.go": w59}, want: ref, suffix: remedy},
		{name: "shown_by_a_read", ledger: map[string]Seen{"f.go": w([][2]int{{9, 9}}, [2]int{5, 9})}, want: "ok"},
		{name: "forced_after_external_edit", ledger: map[string]Seen{"f.go": {SHA: olderSHA, Written: true, Spans: [][2]int{{5, 9}}, Shown: [][2]int{{9, 9}}}}, opt: Options{Force: true}, want: ref},
		{name: "forced_nothing_shown", ledger: map[string]Seen{"f.go": w59}, opt: Options{Force: true}, want: ref},
		{name: "quiet_whole_merged", ledger: map[string]Seen{"f.go": {SHA: sha, Written: true}}, want: ref},
		{name: "quiet_ranged_merged", ledger: map[string]Seen{"f.go": w(nil, [2]int{5, 5}, [2]int{6, 9})}, want: ref},
		// The second quoted write changed line 5 at a new sha and left w|5-5 with nothing shown, then a ranged --quiet licence of 6-9 joined it.
		{name: "later_one_line_write", ledger: map[string]Seen{"f.go": w(nil, [2]int{5, 5}, [2]int{6, 9})}, want: ref},
		{name: "plan_licence_later_call", ledger: map[string]Seen{"f.go": {SHA: sha, Written: true, Spans: [][2]int{{6, 9}}}}, want: ref},
		{name: "width_cut", ledger: map[string]Seen{"f.go": w([][2]int{{6, 8}}, [2]int{5, 9})}, want: ref},
		{name: "same_call_rewrite", ledger: map[string]Seen{"f.go": w59}, opt: Options{SameCallRewrite: true, Computed: true}, want: "ok"},
		{name: "quoted_no_kept_tail", ledger: map[string]Seen{"f.go": w59}, opt: Options{Computed: true},
			in: func(t *testing.T, root string) []Input { return planOf(t, root, noTail) }, want: ref},
		{name: "quoted_kept_tail", ledger: map[string]Seen{"f.go": w59}, opt: Options{Computed: true},
			in: func(t *testing.T, root string) []Input { return planOf(t, root, keptTail) }, want: "ok"},
		{name: "alias_quiet_record", alias: true, ledger: map[string]Seen{"g.go": {SHA: sha, Written: true, Spans: [][2]int{{6, 9}}}}, want: ref},
		{name: "alias_read_record", alias: true, ledger: map[string]Seen{"f.go": w59, "g.go": readOf([2]int{6, 9})}, want: "ok"},
		// The plan's own record (a read of 6-8) does not cover line 9, but another spelling's record shows it.
		{name: "alias_shown_by_another_spelling", alias: true, ledger: map[string]Seen{"f.go": w([][2]int{{9, 9}}, [2]int{5, 9}), "g.go": readOf([2]int{6, 8})}, want: "ok"},
		{name: "ends_on_last_line", ledger: map[string]Seen{"f.go": w59}, in: func(*testing.T, string) []Input { return rep(6, 9) }, want: "ok"},
		{name: "read_shaped_record", ledger: map[string]Seen{"f.go": readOf([2]int{1, 9})}, want: "ok"},
		{name: "forced_stale_read_shaped", ledger: map[string]Seen{"f.go": {SHA: olderSHA, Spans: [][2]int{{1, 9}}}}, opt: Options{Force: true}, want: "ok"},
		// A single-line replace of the second match whose first match, line 6, is a line the caller wrote and no read showed.
		{name: "occurrence_after_a_written_match", file: twice, ledger: map[string]Seen{"f.go": {SHA: twiceSHA, Written: true, Spans: [][2]int{{5, 9}}}},
			in: func(*testing.T, string) []Input {
				return []Input{{Path: "f.go", Op: "replace", StartPat: mustRe(`^\t}`), Occurrence: 2, Anchor: "}", Lines: -1, Body: []string{"\t} // x"}}}
			}, want: "have not been read"},
		// Records an mrw before this change wrote for a quoted write: read-shaped spans. The residual window ADR-208 Decision 1a states, pinned so a change to it is seen.
		{name: "legacy_read_shaped_quote_record", ledger: map[string]Seen{"f.go": readOf([2]int{5, 9})}, want: "ok"},
		// That legacy record, then a quoted write that left the sha unchanged: the merge copies the legacy spans into the shown set (seen.go merge).
		{name: "legacy_then_same_sha_t2_write", ledger: map[string]Seen{"f.go": w([][2]int{{5, 9}}, [2]int{5, 9})}, want: "ok"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			content := body
			if c.file != "" {
				content = c.file
			}
			if err := os.WriteFile(filepath.Join(root, "f.go"), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			in := rep(6, 8)
			if c.alias {
				if err := os.Symlink("f.go", filepath.Join(root, "g.go")); err != nil {
					t.Skipf("no symlinks: %v", err)
				}
				in[0].Path = "g.go"
			}
			if c.in != nil {
				in = c.in(t, root)
			}
			opt := c.opt
			opt.Seen = c.ledger
			res, err := Apply(root, in, opt)
			if err != nil {
				t.Fatal(err)
			}
			got, _ := os.ReadFile(filepath.Join(root, "f.go"))
			if c.want == "ok" {
				if !res.Applied || res.Failed != 0 {
					t.Fatalf("refused: %+v", res.Hunks)
				}
				return
			}
			if res.Applied || res.Failed != 1 {
				t.Fatalf("applied %v failed %d, want one refusal: %+v", res.Applied, res.Failed, res.Hunks)
			}
			var reason string
			for _, h := range res.Hunks {
				if h.Status == StatusFailed {
					reason = h.Reason
				}
			}
			if !strings.Contains(reason, c.want) {
				t.Errorf("reason %q lacks %q", reason, c.want)
			}
			if c.suffix != "" && !strings.HasSuffix(reason, c.suffix) {
				t.Errorf("reason %q does not end with %q", reason, c.suffix)
			}
			if string(got) != content {
				t.Errorf("a refused plan changed f.go: %q", got)
			}
		})
	}
}

func mustRe(s string) *regexp.Regexp { return regexp.MustCompile(s) }
