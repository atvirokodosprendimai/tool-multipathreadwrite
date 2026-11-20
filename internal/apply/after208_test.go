package apply

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/ingest"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-208 T1. A multi-line replace on a file whose ledger record is whole and
// written-shaped (mrw wrote it) is refused unless the line after the range was
// SHOWN since the write, in the file's current version: ADR-005's whole-file
// licence of a written file made every line "covered", so the closer the range
// leaves behind went unseen. f.go is func A() { / if x { / } / }, and the plan
// replaces lines 1-3.
func TestAMultiLineReplaceOnAWrittenFileNeedsTheLineAfterShown(t *testing.T) {
	const body = "func A() {\n\tif x {\n\t}\n}\n"
	sha := seen.SHA([]byte(body))
	rep := func(end int) Input {
		return Input{Path: "f.go", Op: "replace", Start: 1, End: end, Anchor: "func A", Lines: -1,
			Body: []string{"func A() {", "\ty()", "}"}}
	}
	written := func(shown ...[2]int) *Seen { return &Seen{SHA: sha, Written: true, Shown: shown} }
	// planOf compiles a foreign document the way the CLI does, to the engine's input.
	planOf := func(t *testing.T, root string, compile func(string, []byte) ([]byte, error), doc string) []Input {
		t.Helper()
		out, err := compile(root, []byte(doc))
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
	const srDoc = "f.go\n<<<<<<< SEARCH\nfunc A() {\n\tif x {\n\t}\n=======\nfunc A() {\n\ty()\n}\n>>>>>>> REPLACE\n"
	const srKept = "f.go\n<<<<<<< SEARCH\nfunc A() {\n\tif x {\n\t}\n}\n=======\nfunc A() {\n\ty()\n}\n>>>>>>> REPLACE\n"
	const apDoc = "*** Begin Patch\n*** Update File: f.go\n@@\n-func A() {\n-\tif x {\n-\t}\n+func A() {\n+\ty()\n+}\n*** End Patch\n"

	cases := []struct {
		name string
		obs  *Seen
		in   func(t *testing.T, root string) []Input
		opt  Options
		want string // "ok", or what the refusal's reason must carry
	}{
		{name: "written_nothing_shown", obs: written(), want: "needs a served line after 3"},
		{name: "written_line_after_shown", obs: written([2]int{4, 4}), want: "ok"},
		{name: "written_ends_on_last_line", obs: written(), in: func(*testing.T, string) []Input { return []Input{rep(4)} }, want: "ok"},
		{name: "read_lines_1_to_4", obs: &Seen{SHA: sha, Spans: [][2]int{{1, 4}}}, want: "ok"},
		{name: "read_lines_1_to_3", obs: &Seen{SHA: sha, Spans: [][2]int{{1, 3}}}, want: "needs a served line after 3"},
		{name: "same_call_rewrite", obs: written(), opt: Options{SameCallRewrite: true, Computed: true}, want: "ok"},
		{name: "later_call_rewrite_plan", obs: written(), opt: Options{Computed: true}, want: "needs a served line after 3"},
		{name: "search_replace_no_kept_tail", obs: written(), opt: Options{Computed: true},
			in: func(t *testing.T, root string) []Input { return planOf(t, root, ingest.CompileSearchReplace, srDoc) }, want: "needs a served line after 3"},
		{name: "search_replace_kept_tail", obs: written(), opt: Options{Computed: true},
			in: func(t *testing.T, root string) []Input { return planOf(t, root, ingest.CompileSearchReplace, srKept) }, want: "ok"},
		{name: "apply_patch", obs: written(), opt: Options{Computed: true},
			in: func(t *testing.T, root string) []Input { return planOf(t, root, ingest.CompileApplyPatch, apDoc) }, want: "needs a served line after 3"},
		{name: "forced_stale_showing", obs: &Seen{SHA: seen.SHA([]byte("older\n")), Written: true, Shown: [][2]int{{4, 4}}}, opt: Options{Force: true}, want: "needs a served line after 3"},
		{name: "forced_fresh_showing", obs: written([2]int{4, 4}), opt: Options{Force: true}, want: "ok"},
		{name: "forced_nothing_shown", obs: written(), opt: Options{Force: true}, want: "needs a served line after 3"},
		{name: "partial_written_licence", obs: &Seen{SHA: sha, Written: true, Spans: [][2]int{{1, 4}}}, want: "ok"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "f.go"), []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
			in := []Input{rep(3)}
			if c.in != nil {
				in = c.in(t, root)
			}
			opt := c.opt
			opt.Seen = map[string]Seen{"f.go": *c.obs}
			res, err := Apply(root, in, opt)
			if err != nil {
				t.Fatal(err)
			}
			got, rerr := os.ReadFile(filepath.Join(root, "f.go"))
			if rerr != nil {
				t.Fatal(rerr)
			}
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
			if c.obs.Written && c.obs.Whole() && !strings.HasSuffix(reason, "mrw read --max-cols 0 f.go:1-4") {
				t.Errorf("reason %q does not end with the read to make", reason)
			}
			if string(got) != body {
				t.Errorf("a refused plan changed f.go: %q", got)
			}
		})
	}
}
