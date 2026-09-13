package authoring

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// tallyPath is where the tally lands for a root, via the same resolver the
// package uses — reading it from the state package rather than rebuilding the
// path keeps the test from asserting a layout ADR-004 owns.
func tallyPath(t *testing.T, root string) string {
	t.Helper()
	dirLine := os.Getenv("XDG_STATE_HOME")
	if dirLine == "" {
		t.Fatal("XDG_STATE_HOME must be pinned by the test, or this writes to the developer's real state")
	}
	matches, err := filepath.Glob(filepath.Join(dirLine, "mrw", "*", "authoring"))
	if err != nil || len(matches) == 0 {
		return ""
	}
	return matches[0]
}

func newRoot(t *testing.T) string {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	return t.TempDir()
}

func TestTallyRoundTripsThroughLoad(t *testing.T) {
	root := newRoot(t)
	if err := Record(root, Applied); err != nil {
		t.Fatal(err)
	}
	got, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if got["applied"] != 1 {
		t.Errorf("applied = %d, want 1 — what was recorded did not come back", got["applied"])
	}
	if got.Plans() != 1 {
		t.Errorf("Plans() = %d, want 1", got.Plans())
	}
}

// The vocabulary must not collapse: six outcomes are six counters, and a
// category is recorded only for the outcome it refines.
func TestTallyCountsEachOutcomeSeparately(t *testing.T) {
	root := newRoot(t)
	for _, o := range []Outcome{Applied, Applied, RefusedApply, CheckNotRun, FailedCheck, RefusedParse} {
		_ = Record(root, o)
	}

	got, _ := Load(root)
	for name, want := range map[string]int{
		"applied": 2, "refused_apply": 1, "check_not_run": 1,
		"failed_check": 1, "refused_parse": 1,
	} {
		if got[name] != want {
			t.Errorf("%s = %d, want %d", name, got[name], want)
		}
	}
	if got.Plans() != 6 {
		t.Errorf("Plans() = %d, want 6 — the denominator counts plans, not counters", got.Plans())
	}
}

// FAIL OPEN. A corrupt tally is discarded, never repaired and never surfaced as
// an error — a caller's next move on "the tally is broken" is the same as on
// "there is no tally yet".
func TestAnUnreadableTallyFailsOpen(t *testing.T) {
	root := newRoot(t)
	if err := Record(root, Applied); err != nil {
		t.Fatal(err)
	}
	p := tallyPath(t, root)
	if p == "" {
		t.Fatal("no tally was written, so this test would pass without exercising anything")
	}
	if err := os.WriteFile(p, []byte("\x00\x01 not a tally\nrefused_parse notanumber\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := Load(root)
	if err != nil {
		t.Errorf("a corrupt tally returned an error: %v — Load must fail open", err)
	}
	if len(got) != 0 {
		t.Errorf("garbage was parsed into %v, want an empty tally", got)
	}
	// And recording after corruption still works, rather than propagating it.
	if err := Record(root, Applied); err != nil {
		t.Errorf("Record after a corrupt tally returned %v, want nil", err)
	}
}

// Record may never fail a write. An unwritable state directory is the cheapest
// way to prove it: the tool must carry on regardless.
func TestRecordNeverFailsAWrite(t *testing.T) {
	root := newRoot(t)
	// A state home that is a FILE, not a directory: MkdirAll under it cannot
	// succeed, so state.Path errors and Record takes its failure path. (A NUL
	// in the value would be rejected by t.Setenv itself and prove nothing.)
	blocker := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_STATE_HOME", blocker)
	if err := Record(root, Applied); err != nil {
		t.Errorf("Record returned %v with an unusable state home; it must never fail a write", err)
	}
}

// ⚠ THE BOUNDARY. This test is ADR-009's Enforced-by.
//
// The tally may hold counts and vocabulary names, and nothing else — no plan
// text, no paths, no anchors, no SHAs. The signature already makes it hard
// (Record takes two typed enums and cannot be handed a string), but a type is
// only a boundary while nobody adds a field, so this reads the written BYTES
// and refuses anything that is not `name count`.
func TestTheTallyNeverRecordsPlanContentOrPaths(t *testing.T) {
	root := newRoot(t)
	// Everything the vocabulary can produce, so the file is as full as it gets.
	for _, o := range []Outcome{Applied, RefusedParse, RefusedApply, CheckNotRun, FailedCheck} {
		_ = Record(root, o)
	}

	p := tallyPath(t, root)
	if p == "" {
		t.Fatal("no tally was written, so this test would pass without exercising anything")
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)

	line := regexp.MustCompile(`^[a-z_]+ [0-9]+$`)
	for _, l := range strings.Split(strings.TrimRight(body, "\n"), "\n") {
		if !line.MatchString(l) {
			t.Errorf("tally line %q is not `name count` — something other than a counter reached the file", l)
			continue
		}
		name := strings.SplitN(l, " ", 2)[0]
		if !names[name] {
			t.Errorf("tally holds %q, which is outside the closed vocabulary", name)
		}
	}

	// And explicitly: none of the things a plan is made of may appear. These are
	// the strings a future "just record the path" change would introduce.
	for _, forbidden := range []string{
		"/", "\\", "@@", ".go", ".txt", "anchor", "sha", "replace", "insert", "delete", "create",
	} {
		if strings.Contains(body, forbidden) {
			t.Errorf("tally contains %q — a plan fragment, path or address reached disk:\n%s", forbidden, body)
		}
	}
}

// ── ADR-055 T2: the recent-window ring ──────────────────────────────────────

// The ring holds the last RecentWindow landed writes and nothing ADR-009
// refuses: three fields per line — unix seconds, the op class, the advisory
// count — no path, no plan text, no address.
func TestRecentKeepsTheLastTenWritesAndNoPaths(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	for i := 0; i < RecentWindow+3; i++ {
		if err := RecordRecent(root, i%2); err != nil {
			t.Fatal(err)
		}
	}
	got := Recent(root)
	if len(got) != RecentWindow {
		t.Fatalf("ring holds %d entries, want %d", len(got), RecentWindow)
	}
	// The oldest three fell off: the survivors are writes 3..12, so the first
	// survivor's advisory bit is 3%2 == 1.
	if got[0].Advisories != 1 || got[len(got)-1].Advisories != (RecentWindow+2)%2 {
		t.Errorf("ring did not keep the LAST %d: %+v", RecentWindow, got)
	}
	p, err := state.Path(root, recentFile)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	// The FILE is bounded, not only the reader's view of it: a ring that
	// trims on read and not on write grows without limit on disk.
	if len(lines) != RecentWindow {
		t.Errorf("ring file holds %d lines, want %d", len(lines), RecentWindow)
	}
	for _, line := range lines {
		f := strings.Fields(line)
		if len(f) != 3 || f[1] != "write" {
			t.Errorf("ring line %q is not `<unix> write <n>`", line)
		}
		if strings.ContainsAny(line, "/.\\") {
			t.Errorf("ring line %q carries something path-shaped", line)
		}
	}
}

// Pattern fires at PatternThreshold advisories in the window and not below it.
// A garbage file reads as empty, never as an error.
func TestPatternFiresAtThreeOfTenAndNotAtTwo(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	for _, n := range []int{0, 1, 0, 1} {
		if err := RecordRecent(root, n); err != nil {
			t.Fatal(err)
		}
	}
	if k, n, fires := Pattern(Recent(root)); fires || k != 2 || n != 4 {
		t.Errorf("two advisories in four: fires=%v k=%d n=%d, want no fire, 2 of 4", fires, k, n)
	}
	if err := RecordRecent(root, 2); err != nil {
		t.Fatal(err)
	}
	if k, n, fires := Pattern(Recent(root)); !fires || k != 3 || n != 5 {
		t.Errorf("three advisories in five: fires=%v k=%d n=%d, want fire, 3 of 5", fires, k, n)
	}

	p, _ := state.Path(root, recentFile)
	if err := os.WriteFile(p, []byte("not a ring\n\x00\n12 write x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := Recent(root); len(got) != 0 {
		t.Errorf("a garbage ring read as %d entries, want 0 (fail open)", len(got))
	}
}

// ── ADR-056 T2: pricing --strict-balance ────────────────────────────────────

// Five counters, `name N` lines only, one outcome per would-refuse write; a
// non-candidate counts nothing; garbage reads as zero; the rate refuses to
// divide by nothing.
func TestPricingCountsCandidatesRefusalsAndOutcomes(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	if _, ok := LoadPricing(root).FalsePositiveRate(); ok {
		t.Error("a fresh root reports a false-positive rate on no evidence")
	}
	seq := []struct {
		candidate, would bool
		outcome          PricingOutcome
	}{
		{true, true, PricedBroke},
		{true, true, PricedUnchecked},
		{true, false, PricedHeld},  // not would-refuse: its outcome is not counted
		{false, false, PricedHeld}, // not a candidate: counts nothing at all
		{true, true, PricedHeld},
	}
	for _, s := range seq {
		if err := RecordPricing(root, s.candidate, s.would, s.outcome); err != nil {
			t.Fatal(err)
		}
	}
	got := LoadPricing(root)
	want := Pricing{Candidates: 4, WouldRefuse: 3, Broke: 1, Held: 1, Unchecked: 1}
	if got != want {
		t.Errorf("pricing = %+v, want %+v", got, want)
	}
	if rate, ok := got.FalsePositiveRate(); !ok || rate != 0.5 {
		t.Errorf("rate = %v ok=%v, want 0.5 true (held 1 of broke+held 2)", rate, ok)
	}

	p, err := state.Path(root, pricingFile)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 5 {
		t.Errorf("pricing file has %d lines, want 5:\n%s", len(lines), raw)
	}
	for _, l := range lines {
		f := strings.Fields(l)
		if len(f) != 2 || !strings.HasPrefix(f[0], "strict_") {
			t.Errorf("pricing line %q is not `strict_<name> N`", l)
		}
		if _, err := strconv.Atoi(f[len(f)-1]); err != nil {
			t.Errorf("pricing line %q does not end in a count", l)
		}
	}

	if err := os.WriteFile(p, []byte("strict_candidates x\n\x00\nnot a counter\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := LoadPricing(root); got != (Pricing{}) {
		t.Errorf("garbage read as %+v, want zero (fail open)", got)
	}
}
