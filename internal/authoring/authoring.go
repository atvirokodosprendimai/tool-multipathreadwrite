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

// Record adds one plan's outcome to the tally.
//
// ⚠ IT NEVER FAILS A WRITE. Every error path returns nil: an unwritable state
// directory, a corrupt tally, a full disk. Measurement that can break the tool
// it measures is worse than no measurement, and this is the rule that keeps a
// counter from becoming load-bearing. The cost of being wrong is a lost count.
func Record(root string, o Outcome) error {
	name := o.name()
	if name == "" {
		return nil // an Outcome outside the vocabulary is not persisted
	}
	t, _ := Load(root) // a corrupt tally is discarded, not repaired
	if t == nil {
		t = Tally{}
	}
	t[name]++
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

// Load reads the tally. It FAILS OPEN: an unreadable or malformed file yields
// an empty tally and no error, because a caller's next move on "the tally is
// broken" is the same as on "there is no tally yet".
func Load(root string) (Tally, error) {
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
