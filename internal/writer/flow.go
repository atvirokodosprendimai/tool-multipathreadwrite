package writer

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/authoring"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/check"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-113: one write, as both surfaces run it. `mrw write` and `mrw_write`
// each prepared a plan, applied it, counted it and checked it in their own
// copy, and the copies drifted: the MCP one ran no check at all. The sequence
// is three phases because the CLI prints its receipt and counts the landing
// BEFORE the check (ADR-072), so a write killed during its check is still
// printed and counted. Each phase returns facts; the surfaces render them.

// CheckMode says whether a write's check runs (ADR-054).
type CheckMode int

const (
	// CheckAuto runs the check when the write landed, touched a file that is
	// not prose, and the tree declares or infers a check.
	CheckAuto CheckMode = iota
	// CheckOff runs none: --no-check, or mrw_write's check: false.
	CheckOff
	// CheckDemand runs it even on prose, and reports a tree with no command
	// as a check that could not run: --check.
	CheckDemand
)

// Request is one write as a surface hands it over: the resolved hunks, the
// apply options (Seen is filled by Land), the check mode and the steps.
type Request struct {
	Root  string
	In    []apply.Input
	Opts  apply.Options
	Check CheckMode
	Steps []check.Step
	// StepFlag is the surface's name for its steps, which a refused step
	// name is reported under: "--then" on the CLI when empty, "then" over MCP,
	// whose caller has no flag to pass (ADR-115).
	StepFlag string
}

// Stage names the gate that refused a write before anything was written.
type Stage int

const (
	// StageHarness — .quality-harness.json could not be read.
	StageHarness Stage = iota + 1
	// StageSteps — a step named no declared step.
	StageSteps
	// StageDepth — a check was due at MRW_STEP_DEPTH's limit (ADR-095).
	StageDepth
	// StageLedger — the ledger could not be loaded.
	StageLedger
)

// Refusal is a write refused before anything was written. It is counted
// refused_apply once, by the phase that returns it; Stage lets a surface say
// it in its own words.
type Refusal struct {
	Stage Stage
	Err   error
}

func (r *Refusal) Error() string { return r.Err.Error() }

// Unwrap returns the gate's own error.
func (r *Refusal) Unwrap() error { return r.Err }

// Prepared is a write that passed every gate that refuses before anything is
// written, carrying the harness it read.
type Prepared struct {
	req Request
	cfg check.Config
}

// Prepare runs the gates: the harness, read only when a check may run or a
// step is asked for (ADR-072: before the commit, so a malformed harness
// writes nothing); the steps resolved to their declared commands (ADR-092);
// and the depth refusal when a check would be due (ADR-095). The steps in req
// are resolved in place.
func Prepare(req Request) (*Prepared, error) {
	p := &Prepared{req: req}
	if (req.Check != CheckOff && !req.Opts.DryRun) || len(req.Steps) > 0 {
		cfg, err := check.Load(req.Root)
		if err != nil {
			return nil, p.refuse(StageHarness, err)
		}
		p.cfg = cfg
	}
	if err := ResolveSteps(p.cfg, req.Steps, req.StepFlag); err != nil {
		return nil, p.refuse(StageSteps, err)
	}
	if req.Check != CheckOff && !req.Opts.DryRun && p.wanted(touchesCode(req.In)) {
		if err := check.DepthRefusal("a check is"); err != nil {
			return nil, p.refuse(StageDepth, err)
		}
	}
	return p, nil
}

func (p *Prepared) refuse(s Stage, err error) error {
	_ = authoring.Record(p.req.Root, authoring.RefusedApply)
	return &Refusal{Stage: s, Err: err}
}

// wanted is ADR-054's rule given whether a write touches a file a check could
// cover: CheckDemand demands it, and otherwise a code path and a command must
// exist. The depth gate and CheckDue share it, so they cannot disagree.
func (p *Prepared) wanted(code bool) bool {
	return p.req.Check == CheckDemand || (code && (p.cfg.Check != "" || p.cfg.ScopedCheck != ""))
}

// Landed is a write that reached Apply: its receipt, and how the landing went.
type Landed struct {
	p *Prepared
	// Res is the apply's receipt.
	Res apply.Result
	// Err is the apply's own error — a filesystem failure, a lock that was
	// not given — or nil. A ledger failure after landing is LedgerErr.
	Err error
	// LedgerErr is the ledger's failure to record a write that landed
	// (writer.LedgerError's cause).
	LedgerErr error
	// Then is every step asked for, not_run, when LedgerErr ended the write
	// before any could follow it (ADR-092 Decision 5).
	Then *check.StepsResult
	// gen is the checkout's write counter as this landing left it (ADR-127),
	// zero when nothing landed or the counter could not move.
	gen int64
}

// Land loads the ledger, applies, and counts the landing (ADR-009, ADR-083,
// ADR-102): a partial commit partially_applied, a refusal or a failed hunk
// refused_apply, a clean dry run nothing, and a landing applied — or, when the
// ledger could not record it and a check was due, check_not_run. A landing
// joins the recent ring (ADR-055); one nothing will check is priced unchecked
// (ADR-056). A ledger that cannot be loaded is a Refusal.
func (p *Prepared) Land() (*Landed, error) {
	root, strict := p.req.Root, p.req.Opts.StrictBalance
	ledger, err := seen.Snapshot(root)
	if err != nil {
		return nil, p.refuse(StageLedger, err)
	}
	opts := p.req.Opts
	opts.Seen = ledger
	res, gen, err := applyCounted(root, p.req.In, opts)
	l := &Landed{p: p, Res: res, gen: gen}
	var lerr *LedgerError
	switch {
	case errors.As(err, &lerr):
		l.LedgerErr = lerr.Err
		_ = authoring.RecordRecent(root, res.Advisories)
		if !strict {
			_ = authoring.RecordPricing(root, res.StrictSingleLine > 0, res.StrictWouldRefuse > 0, authoring.PricedUnchecked)
		}
		if len(p.req.Steps) > 0 {
			l.Then = notRun(p.req.Steps)
		}
		switch {
		case l.CheckDue():
			_ = authoring.Record(root, authoring.CheckNotRun)
		case res.Applied && !res.DryRun:
			_ = authoring.Record(root, authoring.Applied)
		}
	case err != nil:
		l.Err = err
		if MutationOf(res) == Partial {
			_ = authoring.Record(root, authoring.PartiallyApplied)
			_ = authoring.RecordRecent(root, res.Advisories)
			if !strict {
				_ = authoring.RecordPricing(root, res.StrictSingleLine > 0, res.StrictWouldRefuse > 0, authoring.PricedUnchecked)
			}
		} else {
			_ = authoring.Record(root, authoring.RefusedApply)
		}
	default:
		if res.Applied && !res.DryRun {
			_ = authoring.RecordRecent(root, res.Advisories)
		}
		switch {
		case res.Failed > 0:
			_ = authoring.Record(root, authoring.RefusedApply)
		case res.DryRun:
			// ADR-079: a clean dry run landed nothing.
		default:
			_ = authoring.Record(root, authoring.Applied)
		}
	}
	return l, nil
}

// CheckDue says whether this write's check runs: only on a real landing and
// not under CheckOff, and then by ADR-054's rule over what it touched.
func (l *Landed) CheckDue() bool {
	if !l.Res.Applied || l.Res.Failed > 0 || l.p.req.Check == CheckOff {
		return false
	}
	_, code := CheckPaths(l.Res.Files)
	return l.p.wanted(code)
}

// Verified is what followed a clean landing: the check, what changed while
// it ran, and the steps.
type Verified struct {
	// Check is the check's result, or nil when none was due.
	Check *check.Result
	// Drift is each file the write touched that changed while the check ran
	// (ADR-112).
	Drift []string
	// DriftWriters is how many other writes landed in the checkout while the
	// check ran (ADR-127): zero when none did, when no check ran, or when the
	// counter could not be read.
	DriftWriters int
	// Then is every step's verdict, present whenever a step was asked for.
	Then *check.StepsResult
	// CheckErr is why a due check could not run at all; the landing is then
	// check_not_run and no step ran.
	CheckErr error
}

// Verify runs what follows a landing that Land returned with neither Err nor
// LedgerErr: the check when it is due, the drift, and the steps — which follow
// a landed write whose check, when one ran, passed. A check that could not run
// at all moves the count to check_not_run here; every other verdict is
// counted by Settle, which the caller runs once it has rendered the receipt.
func (l *Landed) Verify(ctx context.Context) Verified {
	root, res, cfg, steps := l.p.req.Root, l.Res, l.p.cfg, l.p.req.Steps
	var v Verified
	if l.CheckDue() {
		written, _ := CheckPaths(res.Files)
		// ADR-112: the baseline is what the write left on disk, taken now.
		before := Before(root, res)
		cr, err := check.Run(ctx, root, cfg, written)
		if err != nil {
			_ = authoring.Reclassify(root, authoring.Applied, authoring.CheckNotRun)
			if len(steps) > 0 {
				v.Then = notRun(steps)
			}
			v.CheckErr = err
			return v
		}
		v.Check = &cr
		if cr.Ran {
			v.Drift = Drift(root, before)
			if l.gen > 0 {
				if n := Writes(root) - l.gen; n > 0 {
					v.DriftWriters = int(n)
				}
			}
		}
	}
	if len(steps) > 0 {
		due := res.Applied && (v.Check == nil || v.Check.OK())
		v.Then = RunSteps(ctx, root, cfg, steps, due)
	}
	return v
}

// Settle moves a landing's count to what Verify found — check_not_run for a
// check that did not run, failed_check for one that did not pass or a step
// that stopped the sequence (ADR-092 Decision 7) — and prices the write from
// the check alone (ADR-056). It runs after the caller rendered the receipt
// (ADR-113): a CLI whose --json output failed exits 2 with the count where
// Land left it, as it did before the sequence was shared. Not for a Verified
// carrying CheckErr, which Verify already counted.
func (l *Landed) Settle(v Verified) {
	if v.CheckErr != nil {
		return
	}
	root, res := l.p.req.Root, l.Res
	switch {
	case v.Check != nil && !v.Check.Ran:
		_ = authoring.Reclassify(root, authoring.Applied, authoring.CheckNotRun)
	case v.Check != nil && !v.Check.OK():
		_ = authoring.Reclassify(root, authoring.Applied, authoring.FailedCheck)
	}
	if s, _, ok := StepStop(v.Then); ok && res.Applied && (v.Check == nil || v.Check.OK()) {
		if s.Ran {
			_ = authoring.Reclassify(root, authoring.Applied, authoring.FailedCheck)
		} else {
			_ = authoring.Reclassify(root, authoring.Applied, authoring.CheckNotRun)
		}
	}
	// ADR-056: a flag-on write is not priced — the question is what the flag
	// WOULD have done, and it just did it.
	if res.Applied && !res.DryRun && !l.p.req.Opts.StrictBalance {
		outcome := authoring.PricedUnchecked
		if v.Check != nil && v.Check.Ran {
			outcome = authoring.PricedBroke
			if v.Check.OK() {
				outcome = authoring.PricedHeld
			}
		}
		_ = authoring.RecordPricing(root, res.StrictSingleLine > 0, res.StrictWouldRefuse > 0, outcome)
	}
}

// CheckPaths is the check's working set after a write. Unlinked and rename
// sources are gone, so confine cannot Stat them; their parent directory still
// exists and is what the check can honour. A rename dest is a Written file of
// its own. The source's extension still sets code: a .go renamed to .txt
// removed code.
func CheckPaths(files []apply.FileResult) (paths []string, code bool) {
	seen := map[string]bool{}
	add := func(p string) {
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		paths = append(paths, p)
	}
	for _, f := range files {
		if f.Removed {
			dir := filepath.Dir(f.Path)
			if dir == "" {
				dir = "."
			}
			add(dir)
			code = code || !apply.IsProse(f.Path)
			continue
		}
		if f.Written {
			add(f.Path)
			code = code || !apply.IsProse(f.Path)
		}
	}
	return paths, code
}

// touchesCode says whether a plan names a path a check could cover, judged
// before the apply as CheckPaths judges after it: every hunk path, an unlink
// target and a rename source by their own extension, and every rename
// destination (ADR-095).
func touchesCode(in []apply.Input) bool {
	for _, h := range in {
		if !apply.IsProse(h.Path) {
			return true
		}
		if h.Op == "rename" && len(h.Body) == 1 && !apply.IsProse(h.Body[0]) {
			return true
		}
	}
	return false
}

// ResolveSteps gives each named step its declared command, and refuses a name
// the project did not declare, naming the ones it did (ADR-092), under flag
// ("--then" when empty).
func ResolveSteps(cfg check.Config, steps []check.Step, flag string) error {
	if flag == "" {
		flag = "--then"
	}
	var cmds map[string]string
	read := false
	for i := range steps {
		if steps[i].AdHoc {
			continue
		}
		// The block is read only now, when a step is asked for by name: a
		// write that asks for none never reads it (ADR-092 T5).
		if !read {
			var err error
			if cmds, err = cfg.StepCommands(); err != nil {
				return err
			}
			read = true
		}
		c, ok := cmds[steps[i].Name]
		if ok {
			steps[i].Command = c
			continue
		}
		names := make([]string, 0, len(cmds))
		for n := range cmds {
			names = append(names, Shown(n))
		}
		sort.Strings(names)
		declared := "none are declared"
		if len(names) > 0 {
			declared = "declared: " + strings.Join(names, ", ")
		}
		return fmt.Errorf("%s %s: .quality-harness.json \"steps\" has no such step (%s)", flag, Shown(steps[i].Name), declared)
	}
	return nil
}

// Shown is s as the caller should see it on a terminal: as written, or quoted
// when it holds a byte a terminal would act on — a step name or command must
// not clear the screen or forge the declared list (ADR-092 T5).
func Shown(s string) string {
	if q := strconv.Quote(s); q[1:len(q)-1] != s {
		return q
	}
	return s
}

// RunSteps runs the sequence when it is due, and otherwise reports every step
// not_run: a plan that did not land, or a check that did not pass, verified
// nothing for a step to follow.
func RunSteps(ctx context.Context, root string, cfg check.Config, steps []check.Step, due bool) *check.StepsResult {
	if due {
		r := check.RunSteps(ctx, root, cfg, steps)
		return &r
	}
	return notRun(steps)
}

func notRun(steps []check.Step) *check.StepsResult {
	r := check.StepsResult{Steps: make([]check.StepResult, len(steps))}
	for i, s := range steps {
		r.Steps[i] = check.StepResult{Step: s, Status: check.StepNotRun, ExitCode: -1}
	}
	return &r
}

// StepStop is the step that stopped the sequence, if one did: the first that
// did not pass and was not skipped.
func StepStop(r *check.StepsResult) (check.StepResult, int, bool) {
	if r == nil {
		return check.StepResult{}, 0, false
	}
	for i, s := range r.Steps {
		if s.Status != check.StepPass && s.Status != check.StepNotRun {
			return s, i, true
		}
	}
	return check.StepResult{}, 0, false
}
