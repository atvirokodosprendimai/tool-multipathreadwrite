// Package authoring counts what happens to the plans mrw is given.
//
// mrw's plan format is bespoke: no model has it in training data, and until
// this package existed nothing measured whether the thing meant to write one
// can. scripts/measure.sh publishes byte and round-trip savings that are all
// conditional on the plan being authored correctly, and scripts/contract.sh
// tests what mrw does WITH a plan — the step before that was unmeasured.
// ADR-009 records the decision and the criterion.
//
// ⚠ WHAT THIS FILE MAY NEVER HOLD. Counts and names from the closed vocabulary
// below. No plan text, no file paths, no anchors, no SHAs, no command lines.
// The boundary is enforced by the SIGNATURE — Record takes ONE typed enum and
// cannot be handed a string — and asserted by
// TestTheTallyNeverRecordsPlanContentOrPaths, which reads the written bytes and
// refuses anything outside `name count`. A tally is a standing temptation to
// record "just the path"; the type is what makes refusing it free.
//
// Nothing here is ever transmitted. It lives in the per-checkout state
// directory ADR-004 defines, beside the ledger, and `mrw stats` is the only
// reader.
package authoring

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// file is the tally's name inside the state directory.
const file = "authoring"

// Outcome is what became of one plan, and it is exactly what cmd/mrw already
// computes to choose an exit status — a projection of a decision made, never a
// second opinion about it. A closed set: the type is the boundary, so a caller
// cannot smuggle a path in as a "category".
type Outcome int

const (
	// Applied — every hunk landed.
	Applied Outcome = iota
	// RefusedParse — the document did not parse. THE outcome this package
	// exists to count: it is the only one that says the FORMAT was the
	// problem rather than the caller's picture of a file.
	RefusedParse
	// RefusedApply — the document parsed and at least one hunk did not apply:
	// a guard that did not hold, a file not read, a path outside the root.
	//
	// ONE bucket, not three, and that is a finding rather than a shortcut.
	// apply.HunkResult.Reason is a free-form string and mrw has no typed error
	// kinds, so splitting this would mean matching on message text that
	// changes — a SECOND opinion about what happened, beside the one the exit
	// status already carries. ADR-009-T1's Stop Condition names exactly that.
	RefusedApply
	// CheckNotRun — written, but no check could run (exit 2).
	CheckNotRun
	// FailedCheck — written, then --check failed (ADR-003 exit 3).
	FailedCheck
)

// names is the ONLY vocabulary written to disk. A counter whose name is not
// here cannot be persisted, which is how the boundary stays a fact rather than
// a habit.
var names = map[string]bool{
	"applied": true, "refused_parse": true, "refused_apply": true,
	"check_not_run": true, "failed_check": true,
}

func (o Outcome) name() string {
	switch o {
	case Applied:
		return "applied"
	case RefusedParse:
		return "refused_parse"
	case RefusedApply:
		return "refused_apply"
	case CheckNotRun:
		return "check_not_run"
	case FailedCheck:
		return "failed_check"
	}
	return ""
}

// Tally is the counts, keyed by the on-disk name.
type Tally map[string]int

// Plans is how many plans the tally has seen — the DENOMINATOR. A rate without
// it is the form that gets quoted out of the population it was measured on,
// which ADR-009 refuses.
func (t Tally) Plans() int {
	n := 0
	for _, o := range []Outcome{Applied, RefusedParse, RefusedApply, CheckNotRun, FailedCheck} {
		n += t[o.name()]
	}
	return n
}

// Vocabulary returns every counter name in the closed vocabulary, in the order
// a reader meets the exits they project (0, 2, 1, 2, 3). It is the list a
// renderer iterates when it must print a name at zero: Names() walks the keys
// PRESENT in a tally, so a counter that never incremented has no key and
// cannot print — which is how a checkout with three broken trees read as
// `applied 96.9%` and nothing else (ADR-054). Not a sixth name: this is the
// same five `names` holds.
func Vocabulary() []string {
	return []string{"applied", "refused_parse", "refused_apply", "check_not_run", "failed_check"}
}

// Landed is how many plans WROTE the tree, whatever happened next: applied,
// plus failed_check (written, then the check failed — exit 3), plus
// check_not_run (written, and no check could run — exit 2). It is NOT "wrote
// and was checked": --no-check and a prose-only plan both record applied and
// sit here as successes. A reader who takes failed_check over Landed as "of
// those we verified" misreads the rate in the direction that flatters the
// tool, which is why the derived line names the denominator (ADR-054).
func (t Tally) Landed() int {
	return t["applied"] + t["failed_check"] + t["check_not_run"]
}

// Names returns the recorded counter names, sorted, so a caller renders a
// stable order without knowing the vocabulary.
func (t Tally) Names() []string {
	out := make([]string, 0, len(t))
	for k := range t {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// record adds one plan's outcome to the tally.
//
// ⚠ IT NEVER FAILS A WRITE. Every error path returns nil: an unwritable state
// directory, a corrupt tally, a full disk. Measurement that can break the tool
// it measures is worse than no measurement, and this is the rule that keeps a
// counter from becoming load-bearing. The cost of being wrong is a lost count.
func record(root string, o Outcome) error {
	name := o.name()
	if name == "" {
		return nil // an Outcome outside the vocabulary is not persisted
	}
	t, _ := load(root) // a corrupt tally is discarded, not repaired
	if t == nil {
		t = Tally{}
	}
	t[name]++
	return save(root, t)
}

// reclassify moves one plan from outcome from to outcome to (ADR-072). The
// write path counts a landing as Applied BEFORE its check runs, so a write
// killed during the check is still counted as landed; the check's verdict then
// moves that one count. It never adds a plan: from is decremented, at a floor
// of zero, as to is incremented.
func reclassify(root string, from, to Outcome) error {
	a, b := from.name(), to.name()
	if a == "" || b == "" {
		return nil
	}
	t, _ := load(root)
	if t == nil {
		t = Tally{}
	}
	if t[a] > 0 {
		t[a]--
	}
	t[b]++
	return save(root, t)
}

// save writes the tally, vocabulary names only, and swallows every error:
// measurement that can break the tool it measures is worse than none.
func save(root string, t Tally) error {
	p, err := state.Path(root, file)
	if err != nil {
		return nil
	}
	var b strings.Builder
	for _, k := range t.Names() {
		if !names[k] {
			continue // never persist a name outside the vocabulary
		}
		fmt.Fprintf(&b, "%s %d\n", k, t[k])
	}
	_ = os.WriteFile(p, []byte(b.String()), 0o600)
	return nil
}

// load reads the tally. It FAILS OPEN: an unreadable or malformed file yields
// an empty tally and no error, because a caller's next move on "the tally is
// broken" is the same as on "there is no tally yet".
func load(root string) (Tally, error) {
	t := Tally{}
	p, err := state.Path(root, file)
	if err != nil {
		return t, nil
	}
	f, err := os.Open(p)
	if err != nil {
		return t, nil
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		k, v, ok := strings.Cut(strings.TrimSpace(sc.Text()), " ")
		if !ok || !names[k] {
			continue // a line outside the vocabulary is discarded
		}
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			continue
		}
		t[k] += n
	}
	return t, nil
}

// reset empties the tally. Unlike Record it DOES report failure: a caller who
// asked to discard their counts needs to know it did not happen, where a
// caller who merely wrote a plan does not need the tally's problems.
func reset(root string) error {
	p, err := state.Path(root, file)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ── ADR-055: the recent-window ring ─────────────────────────────────────────

// recentFile is the ring's name inside the state directory, beside the tally.
const recentFile = "recent"

// RecentWindow is how many landed writes the ring keeps. By count, not by
// clock: a session is what the caller is doing, not a time of day.
const RecentWindow = 10

// PatternThreshold is how many of the window's writes must carry an advisory
// before the receipt says so. Three is the point at which "again" is a
// pattern and not a coincidence.
const PatternThreshold = 3

// RecentEntry is one landed write: when, what class, how many advisories.
// Nothing ADR-009 refuses — no path, no plan text, no address.
type RecentEntry struct {
	Unix       int64
	Op         string
	Advisories int
}

// recordRecent appends one landed write to the ring and trims it to
// RecentWindow. Like Record it never fails a write: every error path returns
// nil and the cost of being wrong is a lost entry.
func recordRecent(root string, advisories int) error {
	entries := recent(root)
	entries = append(entries, RecentEntry{Unix: time.Now().Unix(), Op: "write", Advisories: advisories})
	if len(entries) > RecentWindow {
		entries = entries[len(entries)-RecentWindow:]
	}
	p, err := state.Path(root, recentFile)
	if err != nil {
		return nil
	}
	var b strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&b, "%d %s %d\n", e.Unix, e.Op, e.Advisories)
	}
	_ = os.WriteFile(p, []byte(b.String()), 0o600)
	return nil
}

// recent reads the ring, oldest first. It FAILS OPEN like Load: an absent,
// unreadable or malformed file is an empty ring, and a torn line is skipped.
func recent(root string) []RecentEntry {
	p, err := state.Path(root, recentFile)
	if err != nil {
		return nil
	}
	f, err := os.Open(p)
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []RecentEntry
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) != 3 || fields[1] != "write" {
			continue // a torn or foreign line is skipped, never an error
		}
		unix, err1 := strconv.ParseInt(fields[0], 10, 64)
		n, err2 := strconv.Atoi(fields[2])
		if err1 != nil || err2 != nil || n < 0 {
			continue
		}
		out = append(out, RecentEntry{Unix: unix, Op: fields[1], Advisories: n})
	}
	if len(out) > RecentWindow {
		out = out[len(out)-RecentWindow:]
	}
	return out
}

// Pattern reports how many of the given entries carried an advisory, how many
// entries there are, and whether that meets PatternThreshold.
func Pattern(entries []RecentEntry) (advisory, total int, fires bool) {
	for _, e := range entries {
		if e.Advisories > 0 {
			advisory++
		}
	}
	return advisory, len(entries), advisory >= PatternThreshold
}

// PatternInfo is the recent-window pattern as a receipt field (ADR-056): how
// many of the window's writes carried an advisory, how many the window
// holds, and whether that meets PatternThreshold. Present on every receipt,
// both transports, so a caller that reads keys sees the key on a quiet day.
type PatternInfo struct {
	AdvisoryWrites int  `json:"advisory_writes"`
	Window         int  `json:"window"`
	Fires          bool `json:"fires"`
}

// PatternOf reads the ring for root and renders it as a PatternInfo.
func PatternOf(root string) PatternInfo {
	k, n, fires := Pattern(Recent(root))
	return PatternInfo{AdvisoryWrites: k, Window: n, Fires: fires}
}

// ── ADR-056: pricing --strict-balance ───────────────────────────────────────

// pricingFile is the counters file beside the tally and the ring.
const pricingFile = "pricing"

// PricingOutcome is how a would-refuse write ended: the check ran and failed
// (the flag would have prevented a broken tree), the check ran and passed (a
// false positive), or no check ran (neither — MCP, --no-check, no command).
type PricingOutcome string

// The three outcomes. Exactly one is recorded per would-refuse write.
const (
	PricedBroke     PricingOutcome = "broke"
	PricedHeld      PricingOutcome = "held"
	PricedUnchecked PricingOutcome = "unchecked"
)

// Pricing is the five counters the BACKLOG pre-registration reads. Counts
// only — nothing ADR-009 refuses.
type Pricing struct {
	Candidates  int `json:"strict_candidates"`
	WouldRefuse int `json:"strict_would_refuse"`
	Broke       int `json:"strict_would_refuse_broke"`
	Held        int `json:"strict_would_refuse_held"`
	Unchecked   int `json:"strict_would_refuse_unchecked"`
}

// recordPricing counts one landed flag-off write. A write with no single-line
// code replace is not a candidate and counts nothing. Like Record it never
// fails the write.
func recordPricing(root string, candidate, wouldRefuse bool, outcome PricingOutcome) error {
	if !candidate {
		return nil
	}
	p := loadPricing(root)
	p.Candidates++
	if wouldRefuse {
		p.WouldRefuse++
		switch outcome {
		case PricedBroke:
			p.Broke++
		case PricedHeld:
			p.Held++
		default:
			p.Unchecked++
		}
	}
	path, err := state.Path(root, pricingFile)
	if err != nil {
		return nil
	}
	var b strings.Builder
	for _, kv := range p.lines() {
		fmt.Fprintf(&b, "%s %d\n", kv.name, kv.n)
	}
	_ = os.WriteFile(path, []byte(b.String()), 0o600)
	return nil
}

// lines is the file order: fixed, so a diff of two pricing files reads.
func (p Pricing) lines() []struct {
	name string
	n    int
} {
	return []struct {
		name string
		n    int
	}{
		{"strict_candidates", p.Candidates},
		{"strict_would_refuse", p.WouldRefuse},
		{"strict_would_refuse_broke", p.Broke},
		{"strict_would_refuse_held", p.Held},
		{"strict_would_refuse_unchecked", p.Unchecked},
	}
}

// loadPricing reads the counters. It fails open: absent or garbage is zero.
func loadPricing(root string) Pricing {
	var p Pricing
	path, err := state.Path(root, pricingFile)
	if err != nil {
		return p
	}
	f, err := os.Open(path)
	if err != nil {
		return p
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) != 2 {
			continue
		}
		n, err := strconv.Atoi(fields[1])
		if err != nil || n < 0 {
			continue
		}
		switch fields[0] {
		case "strict_candidates":
			p.Candidates = n
		case "strict_would_refuse":
			p.WouldRefuse = n
		case "strict_would_refuse_broke":
			p.Broke = n
		case "strict_would_refuse_held":
			p.Held = n
		case "strict_would_refuse_unchecked":
			p.Unchecked = n
		}
	}
	return p
}

// FalsePositiveRate is held / (broke + held); ok is false when nothing
// would-refused was checked, so a caller never divides by zero or reports
// 0% on no evidence.
func (p Pricing) FalsePositiveRate() (rate float64, ok bool) {
	checked := p.Broke + p.Held
	if checked == 0 {
		return 0, false
	}
	return float64(p.Held) / float64(checked), true
}

// ── ADR-079: one lock for the tally, the ring and the pricing counters ─────

// lockName is the tally's lock, one for the three files. Each exported call
// takes it once, for its own file: a write's tally, ring and pricing updates are
// three holds, and `mrw stats` reads the three in three — each file whole, not
// the three at one instant (the review of #240).
const lockName = file + ".lock"

// locked runs fn holding the tally's lock, and does not run it when the lock
// cannot be taken, saying why. Every file here is rewritten by
// truncate-and-write, so a writer racing another unlocked could read one
// emptied, rebuild from nothing and save — the whole tally lost, not a count. A
// writer that cannot lock therefore writes nothing, and the tally loses that one
// count, which it already accepts: a lock file mrw cannot open says nothing
// about whether the tally beside it is writable (the reviews of #240). Nothing
// here calls locked from inside locked.
func locked(root string, fn func()) error {
	release, err := state.Hold(root, lockName)
	if err != nil {
		return err
	}
	defer release()
	fn()
	return nil
}

// lockedRead runs fn under the tally's lock, or without it when the lock cannot
// be taken: a report read unlocked may meet a file mid-rewrite and show it empty,
// but it writes nothing back, so it cannot lose what it misreads.
func lockedRead(root string, fn func()) {
	if locked(root, fn) != nil {
		fn()
	}
}

// Record adds one plan's outcome to the tally, under its lock. It never fails a
// write: every error is swallowed, a lock it cannot take included, and the cost
// of being wrong is a lost count.
func Record(root string, o Outcome) error {
	_ = locked(root, func() { _ = record(root, o) })
	return nil
}

// Reclassify moves one plan from outcome from to outcome to (ADR-072), under
// the tally's lock. It never adds a plan and never fails a write.
func Reclassify(root string, from, to Outcome) error {
	_ = locked(root, func() { _ = reclassify(root, from, to) })
	return nil
}

// Load reads the tally under its lock, or past a lock it cannot take. It fails
// open: an unreadable or malformed file is an empty tally and no error.
func Load(root string) (t Tally, err error) {
	lockedRead(root, func() { t, err = load(root) })
	return t, err
}

// Reset empties the tally under its lock and returns what it discarded, read
// under the same hold: read before it, a count recorded in between was
// discarded unreported (the review of #240). It reports a failure, a lock it
// cannot take included: a caller who asked to discard their counts needs to
// know it did not happen.
func Reset(root string) (discarded Tally, err error) {
	if lerr := locked(root, func() {
		discarded, _ = load(root)
		err = reset(root)
	}); lerr != nil {
		return nil, lerr
	}
	return discarded, err
}

// RecordRecent appends one landed write to the ring (ADR-055), under the
// tally's lock. It never fails a write.
func RecordRecent(root string, advisories int) error {
	_ = locked(root, func() { _ = recordRecent(root, advisories) })
	return nil
}

// Recent reads the ring under the tally's lock, oldest first; it fails open.
func Recent(root string) (entries []RecentEntry) {
	lockedRead(root, func() { entries = recent(root) })
	return entries
}

// RecordPricing counts one landed flag-off write (ADR-056), under the tally's
// lock. It never fails a write.
func RecordPricing(root string, candidate, wouldRefuse bool, outcome PricingOutcome) error {
	_ = locked(root, func() { _ = recordPricing(root, candidate, wouldRefuse, outcome) })
	return nil
}

// LoadPricing reads the counters under the tally's lock; it fails open.
func LoadPricing(root string) (p Pricing) {
	lockedRead(root, func() { p = loadPricing(root) })
	return p
}
