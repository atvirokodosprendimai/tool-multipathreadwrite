package read

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

func TestParseSpec(t *testing.T) {
	for _, tc := range []struct {
		in     string
		path   string
		ranges int
	}{
		{in: "a.go", path: "a.go", ranges: 0},
		{in: "a.go:5", path: "a.go", ranges: 1},
		{in: "a.go:1-8,100-130", path: "a.go", ranges: 2},
		{in: "a.go:3-", path: "a.go", ranges: 1},
		{in: "a.go:/func Foo/,/^}/", path: "a.go", ranges: 1},
		{in: "a.go:/x/,3-9", path: "a.go", ranges: 2},
	} {
		got, err := ParseSpec(tc.in)
		if err != nil {
			t.Errorf("ParseSpec(%q): %v", tc.in, err)
			continue
		}
		if got.Path != tc.path || len(got.Ranges) != tc.ranges {
			t.Errorf("ParseSpec(%q) = path %q, %d range(s); want %q, %d",
				tc.in, got.Path, len(got.Ranges), tc.path, tc.ranges)
		}
	}
	for _, bad := range []string{"a.go:", "a.go:x", "a.go:9-3", "a.go:/[/"} {
		if _, err := ParseSpec(bad); err == nil {
			t.Errorf("ParseSpec(%q) succeeded, want error", bad)
		}
	}
}

func fixture(t *testing.T) (string, Options) {
	t.Helper()
	root := t.TempDir()
	body := "package p\n\nfunc Foo() {\n\tdoThing()\n}\n\nfunc Bar() {\n\treturn\n}\n"
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, Options{Numbers: true}
}

func run(t *testing.T, root string, opt Options, specs ...string) (string, int) {
	t.Helper()
	var parsed []Spec
	for _, s := range specs {
		sp, err := ParseSpec(s)
		if err != nil {
			t.Fatal(err)
		}
		parsed = append(parsed, sp)
	}
	var sb strings.Builder
	_, n := Run(&sb, root, parsed, opt)
	return sb.String(), n
}

func TestNumericRangesAndHeader(t *testing.T) {
	root, opt := fixture(t)
	out, problems := run(t, root, opt, "a.go:3-5")
	if problems != 0 {
		t.Fatalf("problems=%d\n%s", problems, out)
	}
	if !strings.Contains(out, "==> a.go  9L") {
		t.Errorf("header missing the line count:\n%s", out)
	}
	if !strings.Contains(out, "@@ 3-5") || !strings.Contains(out, "    3| func Foo() {") {
		t.Errorf("range not rendered:\n%s", out)
	}
	if strings.Contains(out, "package p") {
		t.Errorf("emitted lines outside the range:\n%s", out)
	}
}

func TestRegexpRange(t *testing.T) {
	root, opt := fixture(t)
	out, problems := run(t, root, opt, "a.go:/func Bar/,/^}/")
	if problems != 0 {
		t.Fatalf("problems=%d\n%s", problems, out)
	}
	if !strings.Contains(out, "@@ 7-9") {
		t.Errorf("wrong span:\n%s", out)
	}
}

// A pattern that matches nothing must say so and be counted. Silence here is
// indistinguishable from an empty file, which is the failure to avoid.
func TestUnmatchedPatternIsReported(t *testing.T) {
	root, opt := fixture(t)
	out, problems := run(t, root, opt, "a.go:/nosuchthing/")
	if problems != 1 {
		t.Errorf("problems=%d, want 1\n%s", problems, out)
	}
	if !strings.Contains(out, "no match") {
		t.Errorf("missing diagnostic:\n%s", out)
	}
}

func TestOverlappingRangesAreMergedNotRepeated(t *testing.T) {
	root, opt := fixture(t)
	out, _ := run(t, root, opt, "a.go:1-4,3-6")
	if strings.Count(out, "@@ ") != 1 || !strings.Contains(out, "@@ 1-6") {
		t.Errorf("ranges not merged:\n%s", out)
	}
	if strings.Count(out, "func Foo() {") != 1 {
		t.Errorf("a line was emitted twice:\n%s", out)
	}
}

// Ranges given out of order must come back ordered and merged. Ascending input
// hid a bug here for a while: the old hand-rolled insertion sort was correct but
// O(n^2), and only unsorted input made that visible (1.83s at 30,000 descending
// ranges, 0.03s after moving to sort.Slice).
func TestDescendingRangesAreSortedAndMerged(t *testing.T) {
	root, opt := fixture(t)
	out, problems := run(t, root, opt, "a.go:9,7-8,3-4,1")
	if problems != 0 {
		t.Fatalf("problems=%d\n%s", problems, out)
	}
	// 7-8 and 9 are adjacent, so they merge into 7-9.
	for _, want := range []string{"@@ 1-1", "@@ 3-4", "@@ 7-9"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	if i, j := strings.Index(out, "@@ 1-1"), strings.Index(out, "@@ 7-9"); i > j {
		t.Errorf("spans emitted out of order:\n%s", out)
	}
}

// The merge must stay correct at a size where an O(n^2) sort would be visible,
// and the result must be exactly one span when every range overlaps its
// neighbour. This is a correctness test, not a benchmark: it would still pass
// slowly, and a timing assertion on CI is a flake waiting to happen.
func TestManyUnsortedRangesMergeCorrectly(t *testing.T) {
	const n = 20000
	in := make([]span, 0, n)
	for i := n; i > 0; i-- { // descending: worst case for insertion sort
		in = append(in, span{start: i, end: i + 1})
	}
	got := merge(in)
	if len(got) != 1 {
		t.Fatalf("got %d spans, want 1 (every range touches its neighbour)", len(got))
	}
	if got[0] != (span{1, n + 1}) {
		t.Errorf("merged span = %+v, want {1 %d}", got[0], n+1)
	}
}

func TestManyFilesOneCall(t *testing.T) {
	root, opt := fixture(t)
	if err := os.WriteFile(filepath.Join(root, "b.go"), []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, problems := run(t, root, opt, "a.go:1", "b.go:2", "gone.go")
	if problems != 1 {
		t.Errorf("problems=%d, want 1 (the missing file)\n%s", problems, out)
	}
	for _, want := range []string{"==> a.go", "==> b.go", "UNREADABLE"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "    2| two") {
		t.Errorf("second file's range missing:\n%s", out)
	}
}

func TestStatAsksForTheFactNotTheArtifact(t *testing.T) {
	root, opt := fixture(t)
	opt.Stat = true
	out, _ := run(t, root, opt, "a.go")
	if strings.Contains(out, "package p") {
		t.Errorf("--stat printed content:\n%s", out)
	}
	if !strings.Contains(out, "sha ") || !strings.Contains(out, "9L") {
		t.Errorf("--stat lost the facts:\n%s", out)
	}
}

// A cap that fires must be visible: a silent truncation reads as the whole file.
func TestMaxLinesReportsWhatItWithheld(t *testing.T) {
	root, opt := fixture(t)
	opt.MaxLines = 3
	out, problems := run(t, root, opt, "a.go")
	if problems == 0 {
		t.Errorf("truncation was not counted as a problem:\n%s", out)
	}
	if !strings.Contains(out, "withheld") {
		t.Errorf("truncation not announced:\n%s", out)
	}
	if strings.Contains(out, "    4|") {
		t.Errorf("emitted more than the cap:\n%s", out)
	}
}

func TestContextAroundSinglePattern(t *testing.T) {
	root, opt := fixture(t)
	opt.Context = 1
	out, _ := run(t, root, opt, "a.go:/doThing/")
	if !strings.Contains(out, "@@ 3-5") {
		t.Errorf("context not applied:\n%s", out)
	}
}

// runObserved is run() plus the ledger observations, which is where the
// licence a read grants actually lives — the printed output is what the caller
// SEES, the observation is what mrw will later let them EDIT, and those two
// went out of step.
func runObserved(t *testing.T, root string, opt Options, specs ...string) (map[string]seen.Observation, int) {
	t.Helper()
	var parsed []Spec
	for _, s := range specs {
		sp, err := ParseSpec(s)
		if err != nil {
			t.Fatal(err)
		}
		parsed = append(parsed, sp)
	}
	var sb strings.Builder
	return Run(&sb, root, parsed, opt)
}

// A ranged read that serves NOTHING must license nothing.
//
// seen.Observation already draws the distinction — a nil Spans is "the whole
// file", an empty-but-non-nil one is "hashed, and none of it shown" — but Run
// declared `served` as a NIL slice, so a range that printed nothing recorded
// the WHOLE FILE. `mrw read a.go:/nomatch/` served no lines, exited 1, and then
// licensed an edit to a line the caller had never seen: a FAILED read granted
// strictly more than a successful partial one, inverting ADR-005.
//
// Asserting Whole() is the assertion that matters. Both the broken and the
// fixed version print the same thing for these specs — nothing, plus a `!!`
// line — so no assertion on the OUTPUT could tell them apart.
func TestARangeThatMatchesNothingObservesNothing(t *testing.T) {
	root, opt := fixture(t)
	for _, spec := range []string{"a.go:/nomatch/", "a.go:99"} {
		observed, problems := runObserved(t, root, opt, spec)
		if problems == 0 {
			t.Errorf("%s: reported no problem", spec)
		}
		o, ok := observed["a.go"]
		if !ok {
			t.Fatalf("%s: nothing observed at all", spec)
		}
		if o.Whole() {
			t.Errorf("%s: recorded the WHOLE FILE for a range that served nothing", spec)
		}
		if len(o.Spans) != 0 {
			t.Errorf("%s: recorded spans %v, want none", spec, o.Spans)
		}
		if o.Covers(3, 3) {
			t.Errorf("%s: licenses an edit to line 3, which was never shown", spec)
		}
	}
}

// The two controls. Either breaking would trade a permissive bug for a
// restrictive one, and a guard that refuses ordinary work is a guard people
// turn off.
func TestWholeAndPartialObservationsAreUnchanged(t *testing.T) {
	root, opt := fixture(t)

	observed, _ := runObserved(t, root, opt, "a.go")
	if o := observed["a.go"]; !o.Whole() || !o.Covers(3, 3) {
		t.Errorf("a plain read no longer observes the whole file: %+v", o)
	}

	observed, _ = runObserved(t, root, opt, "a.go:3-5")
	o := observed["a.go"]
	if o.Whole() {
		t.Error("a partial read observed the whole file")
	}
	if len(o.Spans) != 1 || o.Spans[0] != [2]int{3, 5} {
		t.Errorf("partial read observed %v, want [[3 5]]", o.Spans)
	}
	if !o.Covers(3, 5) || o.Covers(6, 6) {
		t.Errorf("a partial read licenses the wrong lines: %+v", o)
	}
}

// An empty file cannot satisfy any range, and saying nothing about it reported
// success for a request that served nothing: `read empty.txt:1` exited 0 in
// silence while `read a.go:99` on a real file correctly said so. Same rule,
// and the empty file was the one place it was not applied.
func TestARangeAgainstAnEmptyFileIsReported(t *testing.T) {
	root, opt := fixture(t)
	if err := os.WriteFile(filepath.Join(root, "empty.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	out, problems := run(t, root, opt, "empty.txt:1")
	if problems != 1 {
		t.Errorf("problems=%d, want 1\n%s", problems, out)
	}
	if !strings.Contains(out, "file has 0 lines") {
		t.Errorf("the reason was not reported:\n%s", out)
	}
	observed, _ := runObserved(t, root, opt, "empty.txt:1")
	if o := observed["empty.txt"]; o.Whole() {
		t.Errorf("an unsatisfiable range on an empty file observed the whole file: %+v", o)
	}
}
