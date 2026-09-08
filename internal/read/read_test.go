package read

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"strconv"
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
	opt.MaxLines = intp(3)
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

// `$` is the LAST LINE, not "unbounded". read shared one sentinel (0) between
// `$` and an omitted end, and downstream 0 means unbounded in whichever
// direction it appears — so a bare `a.go:$` resolved to 1-total and served the
// WHOLE file. On a 2,000-line file that is 2,000 lines returned for a one-line
// request, which is the round trip mrw exists to remove, and the ledger then
// recorded every one of them as seen.
//
// The README says "`$` is the last line", and the WRITE path agrees — plan's
// ParseAddr maps `$` to {EOF, EOF} and `@@ f.txt $ replace` touches one line.
// read was the only reader of `$` that disagreed, and nothing asserted it.
//
// Counting the served lines is the assertion that matters: the `@@ 5-5` header
// is derived from the resolved range, so it is fair evidence, but a row that
// checked only the header would not notice content leaking past it.
func TestDollarIsTheLastLineNotTheWholeFile(t *testing.T) {
	root, opt := fixture(t)
	for _, spec := range []string{"a.go:$", "a.go:$-$"} {
		out, problems := run(t, root, opt, spec)
		if problems != 0 {
			t.Fatalf("%s: problems=%d\n%s", spec, problems, out)
		}
		if !strings.Contains(out, "@@ 9-9") {
			t.Errorf("%s did not resolve to the last line:\n%s", spec, out)
		}
		if n := strings.Count(out, "|"); n != 1 {
			t.Errorf("%s served %d lines, want 1:\n%s", spec, n, out)
		}
		if strings.Contains(out, "package p") {
			t.Errorf("%s served the first line of the file:\n%s", spec, out)
		}
	}
}

// The control: `$` as an END was already right, by accident — an omitted end
// and `$` both meant "to EOF" there, so they agreed. It must stay right, or
// the fix has traded one wrong reading of `$` for another.
func TestDollarAsAnEndStillRunsToEOF(t *testing.T) {
	root, opt := fixture(t)
	out, problems := run(t, root, opt, "a.go:7-$")
	if problems != 0 {
		t.Fatalf("problems=%d\n%s", problems, out)
	}
	if !strings.Contains(out, "@@ 7-9") {
		t.Errorf("7-$ did not run to EOF:\n%s", out)
	}
}

// A range written with `$` can only be judged reversed once the file's length
// is known: `$-5` is fine on a five-line file and reversed on a longer one.
// Before this it was neither — `$` became 0, 0 became 1, and `$-5` SERVED
// lines 1-5 at exit 0. Content for an address that cannot be satisfied.
func TestAReversedRangeWrittenWithDollarIsNotServed(t *testing.T) {
	root, opt := fixture(t) // a.go is 9 lines, so $-5 is 9-5
	out, problems := run(t, root, opt, "a.go:$-5")
	if problems != 1 {
		t.Fatalf("problems=%d, want 1\n%s", problems, out)
	}
	if !strings.Contains(out, "ends before it starts") {
		t.Errorf("the reason was not reported:\n%s", out)
	}
	if strings.Contains(out, "|") {
		t.Errorf("served content for an unsatisfiable range:\n%s", out)
	}
}

// MSYS2 rewrites a regex address BEFORE mrw starts, so the README's own quoted
// example fails in Git Bash and the parse error names a line number the caller
// never typed. mrw cannot prevent the mangling — it happens in the process-spawn
// layer — but it can recognise the wreckage and say so (issue #45).
func TestAMangledMSYSSpecSaysWhatHappened(t *testing.T) {
	mangled := `cmd\mrw\main.go;C:\Users\me\AppData\Local\Programs\Git\^func main\`
	_, err := ParseSpec(mangled)
	if err == nil {
		t.Fatal("a mangled spec parsed cleanly")
	}
	for _, want := range []string{"MSYS2", "MSYS2_ARG_CONV_EXCL"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error does not mention %s:\n%v", want, err)
		}
	}
}

// And the hint must stay quiet on an ordinary mistake, or it becomes noise on
// every bad spec and stops being read.
func TestAnOrdinaryBadSpecGetsNoMSYSHint(t *testing.T) {
	for _, s := range []string{"a.go:notanumber", "a.go:", "a.go:5-x"} {
		_, err := ParseSpec(s)
		if err == nil {
			continue
		}
		if strings.Contains(err.Error(), "MSYS") {
			t.Errorf("%q got an MSYS hint it should not have:\n%v", s, err)
		}
	}
}

// An absolute path is a whole path, on whichever platform is running. On
// Windows a drive letter carries a colon, and ParseSpec looked for the range
// separator with LastIndex over the entire string — so `C:\dir\f.go` parsed as
// the path "C" with the range `\dir\f.go`, and every absolute Windows path was
// unparseable with a "bad line number" naming most of the path.
//
// Written to be meaningful on both platforms rather than skipped off Windows:
// it builds the platform's OWN absolute path, so on Linux and macOS it pins
// that nothing regressed and on Windows it is the regression itself. Found by
// the windows CI job, 2026-09-03.
func TestAnAbsolutePathIsAWholePathOnThisPlatform(t *testing.T) {
	abs, err := filepath.Abs(filepath.Join("dir", "f.go"))
	if err != nil {
		t.Fatal(err)
	}
	sp, err := ParseSpec(abs)
	if err != nil {
		t.Fatalf("ParseSpec(%q) failed: %v", abs, err)
	}
	if sp.Path != abs {
		t.Errorf("Path = %q, want the whole path %q — the volume's colon was read as a range separator", sp.Path, abs)
	}
	if len(sp.Ranges) != 0 {
		t.Errorf("Ranges = %+v, want none", sp.Ranges)
	}

	// And a range still attaches to an absolute path.
	sp, err = ParseSpec(abs + ":2-4")
	if err != nil {
		t.Fatalf("ParseSpec(%q) failed: %v", abs+":2-4", err)
	}
	if sp.Path != abs || len(sp.Ranges) != 1 {
		t.Errorf("Path = %q with %d range(s), want %q with 1", sp.Path, len(sp.Ranges), abs)
	}
}

// TestAGlobThatTheShellDidNotExpandSaysSo covers ADR-015's other half.
//
// "no such file or directory" for a path holding a `*` is true and answers the
// wrong question. Worse, the shape that produces it most often never reaches
// mrw at all: zsh refuses `dir/*.go:1-3` outright, because the address suffix
// stops the pattern matching any file. Quoting is what a caller tries next, and
// quoting is what produces this message.
func TestAGlobThatTheShellDidNotExpandSaysSo(t *testing.T) {
	root := t.TempDir()
	var b strings.Builder
	sp, err := ParseSpec("sub/*.go:1-3")
	if err != nil {
		t.Fatal(err)
	}
	Run(&b, root, []Spec{sp}, Options{})
	got := b.String()
	if !strings.Contains(got, "UNREADABLE") {
		t.Fatalf("expected an unreadable report, got:\n%s", got)
	}
	for _, want := range []string{"glob", "--grep"} {
		if !strings.Contains(got, want) {
			t.Errorf("the report never mentions %q, so a caller reads it as a missing file:\n%s", want, got)
		}
	}
}

// TestAnOrdinaryMissingFileGetsNoGlobHint is the silence case for the read
// side: a path with no metacharacter is simply missing, and saying anything
// about globs there would be a guess dressed as help.
func TestAnOrdinaryMissingFileGetsNoGlobHint(t *testing.T) {
	root := t.TempDir()
	var b strings.Builder
	sp, err := ParseSpec("nope.go")
	if err != nil {
		t.Fatal(err)
	}
	Run(&b, root, []Spec{sp}, Options{})
	if strings.Contains(b.String(), "glob") {
		t.Errorf("the glob hint fired for a path with no metacharacter:\n%s", b.String())
	}
}

// A relative end is the form a caller arrives with from sed: `A,+N` is the line
// A resolves to plus the N lines after it. Before ADR-026 the comma separated
// two addresses and `+N` parsed as the ABSOLUTE line N, because strconv.Atoi
// accepts a leading sign — so `/func Foo/,+2` served the match and line 2, at
// exit 0, and the receipt was the only place that said so.
//
// Asserting the served span is the assertion that matters: the old reading and
// the new one both exit 0 and both print lines, so only WHICH lines tells them
// apart.
func TestARelativeEndServesTheLinesAfterTheStart(t *testing.T) {
	root, opt := fixture(t)
	for _, spec := range []string{"a.go:/func Foo/,+2", "a.go:3,+2"} {
		out, problems := run(t, root, opt, spec)
		if problems != 0 {
			t.Fatalf("%s: problems=%d\n%s", spec, problems, out)
		}
		if !strings.Contains(out, "@@ 3-5") {
			t.Errorf("%s did not serve 3-5:\n%s", spec, out)
		}
		if n := strings.Count(out, "|"); n != 3 {
			t.Errorf("%s served %d lines, want 3:\n%s", spec, n, out)
		}
		if strings.Contains(out, "package p") {
			t.Errorf("%s served line 1, so the suffix was read as an absolute address:\n%s", spec, out)
		}
	}
	// The ledger is the half a caller cannot see: the licence must cover
	// exactly the lines served, or ADR-002's per-line guard is being granted
	// for lines nobody was shown.
	obs, problems := runObserved(t, root, opt, "a.go:/func Foo/,+2")
	if problems != 0 {
		t.Fatalf("problems=%d", problems)
	}
	o, ok := obs["a.go"]
	if !ok {
		t.Fatal("a.go was not observed at all")
	}
	if !o.Covers(3, 5) {
		t.Errorf("the observation does not cover 3-5, so the served lines were not licensed: %+v", o)
	}
	if o.Covers(6, 6) {
		t.Errorf("the observation covers line 6, which was never served: %+v", o)
	}
}

// An end past the last line CLAMPS, which is not a new rule: `2-99` on this
// fixture already serves 2-9 and exits 0. A relative end that refused instead
// would make the same overrun mean two different things depending on how it
// was written.
func TestARelativeEndPastTheLastLineClamps(t *testing.T) {
	root, opt := fixture(t)
	out, problems := run(t, root, opt, "a.go:8,+10")
	if problems != 0 {
		t.Fatalf("problems=%d, want 0 — an end past EOF clamps, as 2-99 does\n%s", problems, out)
	}
	if !strings.Contains(out, "@@ 8-9") {
		t.Errorf("did not clamp to the last line:\n%s", out)
	}
}

// `+N` with nothing before it names no start to be relative to. ADR-015: the
// refusal names the fix rather than only the mistake.
func TestARelativeEndWithNothingBeforeItIsRefused(t *testing.T) {
	_, err := ParseSpec("a.go:+3")
	if err == nil {
		t.Fatal("a.go:+3 parsed; a relative end with no start must be refused")
	}
	for _, want := range []string{"+3", "3", "A,+3"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q, so it does not name the fix: %v", want, err)
		}
	}
}

// `,+0` says exactly what `A` alone says, and `0` is already refused as a line
// number. Accepting it would be the one case where a relative end is a no-op,
// which is a thing to explain rather than a thing to allow.
func TestARelativeEndOfZeroIsRefused(t *testing.T) {
	_, err := ParseSpec("a.go:2,+0")
	if err == nil {
		t.Fatal("a.go:2,+0 parsed; a zero relative end must be refused")
	}
	if !strings.Contains(err.Error(), "+0") {
		t.Errorf("the refusal does not name what was written: %v", err)
	}
}

// THE REPRODUCER FROM THE SECOND REVIEW, ON THE PATH IT WAS FOUND ON. The
// existing boundary test builds an apply.Input, so neither read branch was
// reached by it — a regression fixture that describes the defect without
// executing it, which is the failure mode testing.md names. `/two/,+MaxInt`
// printed `@@ 2--9223372036854775807` and served nothing at exit 0.
func TestARelativeEndAtTheIntegerBoundaryClampsOnARead(t *testing.T) {
	root, opt := fixture(t)
	for _, spec := range []string{
		"a.go:/func Foo/,+" + strconv.Itoa(math.MaxInt), // the pattern branch: no end<start net beneath it
		"a.go:3,+" + strconv.Itoa(math.MaxInt),          // the numeric branch, which had one by accident
	} {
		out, problems := run(t, root, opt, spec)
		if problems != 0 {
			t.Errorf("%s: problems=%d, want 0 — a relative end past EOF clamps on a read\n%s", spec, problems, out)
		}
		if !strings.Contains(out, "@@ 3-9") {
			t.Errorf("%s did not clamp to the last line:\n%s", spec, out)
		}
		if strings.Contains(out, "--") {
			t.Errorf("%s produced an inverted span, so the end wrapped:\n%s", spec, out)
		}
	}
}

// -C is the same arithmetic one line away, and the flag refuses only a NEGATIVE
// value, so a caller can reach it. It printed `@@ 1--9223372036854775807`.
func TestContextAtTheIntegerBoundaryClamps(t *testing.T) {
	root, opt := fixture(t)
	out, problems := run(t, root, Options{Numbers: opt.Numbers, Context: math.MaxInt}, "a.go:/func Foo/")
	if problems != 0 {
		t.Fatalf("problems=%d, want 0\n%s", problems, out)
	}
	if !strings.Contains(out, "@@ 1-9") {
		t.Errorf("a huge -C did not clamp to the whole file:\n%s", out)
	}
	if strings.Contains(out, "--") {
		t.Errorf("a huge -C produced an inverted span, so the end wrapped:\n%s", out)
	}
}

// A pattern whose body ends in a literal backslash: the closing slash is NOT
// escaped, because the backslash before it is itself escaped. All three
// scanners tested only the preceding byte and read this as unclosed.
func TestAPatternEndingInABackslashIsClosed(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "b.txt"), []byte("a\\b\nplain\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, problems := run(t, root, Options{Numbers: true}, `b.txt:/\\/,+1`)
	if problems != 0 {
		t.Fatalf("problems=%d, want 0 — a pattern ending in a backslash is closed\n%s", problems, out)
	}
	if !strings.Contains(out, "@@ 1-2") {
		t.Errorf("the pattern did not resolve to its match plus one:\n%s", out)
	}
}

// The read grammar accepted four shapes the plan grammar refused, which is the
// divergence ADR-026 claims to have ended: `/` was an empty regexp matching
// every line at exit 0, `//` the same, `/a/garbage` compiled as the pattern
// `a/garbage`, and `/a/,/b/,/c/` silently became its first endpoint. Found by
// the fourth Codex review of PR #125.
func TestAMalformedPatternAddressIsRefused(t *testing.T) {
	for _, c := range []struct{ addr, names string }{
		{"/", "never closed"},
		{"//", "empty pattern"},
		{"/a/garbage", "after the pattern"},
		{"/a/,/b/,/c/", "after the end pattern"},
		{"/a/,/b", "never closed"},
		// The fifth review's two: an empty END pattern, and a trailing comma
		// whose empty component splitRanges used to drop — `5,+2,` became
		// `5,+2` on the read path while the plan path refused the whole string.
		{"/a/,//", "empty pattern"},
		{"5,+2,", "empty range"},
		{"/a/,/b/,", "empty range"},
	} {
		if _, err := ParseSpec("f.txt:" + c.addr); err == nil {
			t.Errorf("f.txt:%s parsed; the plan path refuses it and the two grammars must agree", c.addr)
		} else if !strings.Contains(err.Error(), c.names) {
			t.Errorf("the refusal of %s does not say %q: %v", c.addr, c.names, err)
		}
	}
	// The controls: the two legal pattern forms still parse.
	for _, ok := range []string{"/a/", "/a/,/b/", "/a/,+2"} {
		if _, err := ParseSpec("f.txt:" + ok); err != nil {
			t.Errorf("f.txt:%s was refused: %v", ok, err)
		}
	}
}

// TestACapOfZeroServesNothing pins ADR-033. `--max-lines 0` used to mean
// UNLIMITED: both guards asked `opt.MaxLines > 0`, so a cap of zero was
// indistinguishable from no cap — and nothing was reported withheld, though the
// README promises whatever is withheld is always reported.
//
// This repository decided the same question the other way twice: body=0 is an
// EMPTY body (ADR-027) and lines=0 is a real assertion about a zero-length span.
func TestACapOfZeroServesNothing(t *testing.T) {
	root := t.TempDir()
	// ⚠ DISTINCT lines. Five copies of "line" let a control that COUNTS
	// occurrences pass on duplicated, substituted or reordered content while
	// claiming the whole file came back (review of PR #133).
	body := "alpha\nbravo\ncharlie\ndelta\necho\n"
	if err := os.WriteFile(filepath.Join(root, "f.txt"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	zero := 0
	var buf bytes.Buffer
	_, problems := Run(&buf, root, []Spec{{Path: "f.txt"}}, Options{Numbers: true, MaxLines: &zero})
	got := buf.String()
	if problems == 0 {
		t.Error("a cap of zero served everything it was asked for, so nothing was reported withheld")
	}
	// ⚠ ANY numbered content line, not the old fixture's word. This checked for
	// "| line" and kept checking for it after the fixture became alpha/bravo/…,
	// so it could have served every line and still passed (review of PR #133).
	if numbered := contentLines(got); len(numbered) != 0 {
		t.Errorf("a cap of zero served %d content line(s):\n%s", len(numbered), got)
	}
	if !strings.Contains(got, "WITHHELD 5 line(s)") {
		t.Errorf("the withholding is not reported with its count, so a caller cannot tell what it did not get:\n%s", got)
	}

	// The control, and the reason this is a pointer rather than a sentinel: an
	// ABSENT cap still serves the whole file. Without this half, "serve nothing
	// always" passes.
	var whole bytes.Buffer
	_, p2 := Run(&whole, root, []Spec{{Path: "f.txt"}}, Options{Numbers: true})
	if p2 != 0 {
		t.Errorf("a read with no cap reported %d problem(s)", p2)
	}
	// ⚠ THE SEQUENCE, compared exactly. Checking each line's PRESENCE and then
	// ordering only the first against the last lets the middle reorder and lets
	// any line repeat — the review of PR #133 reproduced that. The extracted
	// sequence is compared to the fixture's, element for element.
	want := []string{"alpha", "bravo", "charlie", "delta", "echo"}
	got2 := contentLines(whole.String())
	if len(got2) != len(want) {
		t.Fatalf("a read with no cap served %d lines, want %d:\n%s", len(got2), len(want), whole.String())
	}
	for i := range want {
		if got2[i] != want[i] {
			t.Errorf("line %d is %q, want %q — the served sequence is not the file's", i+1, got2[i], want[i])
		}
	}
}

// intp is ADR-033's "a cap is set" in test form: nil means no cap.
func intp(n int) *int { return &n }

// contentLines extracts the served text of every numbered line, in order, so a
// control can compare the SEQUENCE rather than count occurrences or check
// presence — both of which accept duplicated and reordered output.
func contentLines(out string) []string {
	var got []string
	for _, l := range strings.Split(out, "\n") {
		i := strings.Index(l, "| ")
		if i <= 0 {
			continue
		}
		if _, err := strconv.Atoi(strings.TrimSpace(l[:i])); err != nil {
			continue
		}
		got = append(got, l[i+2:])
	}
	return got
}

// ADR-036. `/from/,/to/` meant three different things depending on which path
// resolved it, and two of those differences had no reason. This is the one that
// matters: a read whose end never matched served everything from the start to
// the end of the file and reported success. That is this project's own headline
// failure wearing a different hat — a read that quietly served MORE than the
// address named is exactly as invisible as a write that quietly changed less,
// and the caller then holds line numbers for a span mrw never agreed to.
func TestAPairedPatternWithNoEndIsRefusedRatherThanExtendedToEOF(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "f.txt"),
		[]byte("alpha\nSTART\nbody one\nbody two\nomega\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, problems := run(t, root, Options{Numbers: true}, `f.txt:/^START$/,/^NEVER$/`)
	if problems == 0 {
		t.Fatalf("a paired pattern with no end was served rather than reported:\n%s", out)
	}
	// ⚠ THE ABSENCE IS THE ASSERTION. A build that reports the range AND still
	// serves it passes a message-only check, and serving is the defect.
	for _, line := range []string{"body one", "body two", "omega"} {
		if strings.Contains(out, line) {
			t.Errorf("content past the start was still served (%q):\n%s", line, out)
		}
	}
	// The report has to say WHICH half failed, or the caller re-reads the file
	// to find out whether it was the start or the end.
	//
	// ⚠ NOT `Contains(out, "NEVER")`. The spec text is echoed in every miss
	// report, and the end pattern is part of the spec — so that assertion is
	// satisfied by the OLD message and cannot fail for this mechanism. It was
	// written that way first and the mutant would have survived.
	if !strings.Contains(out, "end pattern") {
		t.Errorf("the report does not say the END was the half that missed:\n%s", out)
	}
}

// The other half of ADR-036, and the smaller one: the end is the first match AT
// OR AFTER the start, which is what the write path already did. An end matching
// the start line closes the span there.
func TestAPairedPatternEndsAtOrAfterItsStart(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "f.txt"),
		[]byte("alpha\nMARK here\nmiddle\nMARK again\nomega\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// The end pattern matches the START line: one line, not a run to the next
	// match. `j > i` served 2-4 here.
	out, problems := run(t, root, Options{Numbers: true}, `f.txt:/^MARK here$/,/MARK/`)
	if problems != 0 {
		t.Fatalf("problems=%d, want 0:\n%s", problems, out)
	}
	if !strings.Contains(out, "@@ 2-2") {
		t.Errorf("an end on the start line did not close the span there:\n%s", out)
	}

	// The control that keeps this a narrowing rather than a ban: a normal later
	// end still spans to it.
	out, problems = run(t, root, Options{Numbers: true}, `f.txt:/^alpha$/,/^middle$/`)
	if problems != 0 {
		t.Fatalf("problems=%d, want 0:\n%s", problems, out)
	}
	if !strings.Contains(out, "@@ 1-3") {
		t.Errorf("a normal paired pattern no longer spans to its end:\n%s", out)
	}

	// The control for the difference ADR-036 KEEPS on purpose: a read still
	// serves a span for EVERY match of the start, where a write refuses unless
	// the start matches exactly once. Dropping that would make this record a
	// different and much larger change.
	//
	// ⚠ The spans must not OVERLAP for this to mean anything. `i = end - 1`
	// advances past the span just served, so a second start INSIDE the first
	// span is skipped by construction and proves nothing about every-start
	// behaviour — an earlier cut of this control asserted exactly that and was
	// red for a reason it had misdiagnosed.
	if err := os.WriteFile(filepath.Join(root, "two.txt"),
		[]byte("START\nx\nEND\ngap\nSTART\ny\nEND\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, problems = run(t, root, Options{Numbers: true}, `two.txt:/^START$/,/^END$/`)
	if problems != 0 {
		t.Fatalf("problems=%d, want 0:\n%s", problems, out)
	}
	// A GAP between them, or the assertion proves nothing: contiguous spans are
	// COALESCED, so 1-3 and 4-6 come back as a single `@@ 1-6` and a test
	// asserting two spans fails for a reason that has nothing to do with
	// every-start behaviour. Measured on the pre-change build.
	if !strings.Contains(out, "@@ 1-3") || !strings.Contains(out, "@@ 5-7") {
		t.Errorf("a start matching twice no longer serves both spans:\n%s", out)
	}
}
