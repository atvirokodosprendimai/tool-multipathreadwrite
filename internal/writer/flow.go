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
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/refusal"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// ADR-113: one write, as both surfaces run it. `mrw write` and `mrw_write`
// each prepared a plan, applied it, counted it and checked it in their own
// copy, and the copies drifted: the MCP one ran no check at all. The sequence
// is three phases because the CLI prints its receipt and counts the landing
// BEFORE the check (ADR-072), so a write killed during its check is still
// printed and counted. Each phase returns facts; the surfaces render them.

// InputFromHunk carries a parsed edit into the engine after the caller has
// resolved any working-set pointer in the target path.
func InputFromHunk(h plan.Hunk, path string) apply.Input {
	return apply.Input{
		Path: path, Start: h.Addr.Start, End: h.Addr.End, Op: string(h.Op),
		StartPat: h.Addr.StartPat, EndPat: h.Addr.EndPat, RelEnd: h.Addr.RelEnd,
		Occurrence: h.Occurrence, CountedBody: h.CountedBody,
		Body: h.Body, SHA: h.SHA, Lines: h.Lines, Anchor: h.Anchor, Quote: h.Quote, SrcLine: h.SrcLine,
		From: h.BodyFrom, FromStart: h.BodyFromAddr.Start, FromEnd: h.BodyFromAddr.End, FromRelEnd: h.BodyFromAddr.RelEnd,
	}
}

// CheckMode says whether a write's check runs (ADR-054, ADR-205).
type CheckMode int

const (
	// CheckOff runs none: the zero value, so a Request that names no mode
	// runs no check, and mrw_write without check: true or `mrw write` without
	// --check (ADR-187, ADR-205).
	CheckOff CheckMode = iota
	// CheckDemand makes a check due after any write that lands, prose
	// included, and reports a tree with no command as a check that could not
	// run: --check, and mrw_write's check: true.
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
	if req.Check != CheckOff && !req.Opts.DryRun {
		if err := check.DepthRefusal("a check is"); err != nil {
			return nil, p.refuse(StageDepth, err)
		}
	}
	return p, nil
}

func (p *Prepared) refuse(s Stage, err error) error {
	authoring.Record(p.req.Root, authoring.RefusedApply)
	return &Refusal{Stage: s, Err: err}
}

// Landed is a write that reached the apply: its receipt, and how the landing went.
type Landed struct {
	p *Prepared
	// Res is the apply's receipt.
	Res apply.Result
	// Err is the apply's own error — a filesystem failure, a lock that was
	// not given — or nil. A ledger failure after landing is LedgerErr.
	Err error
	// LedgerErr is the failure after a landing that ended the write: the ledger's
	// failure to record it (writer.LedgerError's cause), or a sync that failed (SyncError).
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
		var unreadable *seen.UnreadableError
		if errors.As(err, &unreadable) {
			err = fmt.Errorf("the read ledger could not be read, so this call wrote and changed nothing: %w", unreadable.Err)
		}
		return nil, p.refuse(StageLedger, err)
	}
	opts := p.req.Opts
	opts.Seen = ledger
	res, gen, err := applyCounted(root, p.req.In, opts)
	newerLedger(root, &res)
	l := &Landed{p: p, Res: res, gen: gen}
	var lerr *LedgerError
	var serr *SyncError
	// A failure after the tree changed (the ledger's, or a sync's, ADR-210) ends
	// the write alike: the files are written, no check or step follows.
	var landed error
	switch {
	case errors.As(err, &lerr):
		landed = lerr.Err
	case errors.As(err, &serr):
		landed = serr
	}
	switch {
	case landed != nil:
		l.LedgerErr = landed
		authoring.RecordRecent(root, res.Advisories)
		if !strict {
			authoring.RecordPricing(root, res.StrictSingleLine > 0, res.StrictWouldRefuse > 0, authoring.PricedUnchecked)
		}
		if len(p.req.Steps) > 0 {
			l.Then = notRun(p.req.Steps)
		}
		switch {
		case l.CheckDue():
			authoring.Record(root, authoring.CheckNotRun)
		case res.Applied && !res.DryRun:
			authoring.Record(root, authoring.Applied)
		}
	case err != nil:
		l.Err = err
		if MutationOf(res) == Partial {
			authoring.Record(root, authoring.PartiallyApplied)
			authoring.RecordRecent(root, res.Advisories)
			if !strict {
				authoring.RecordPricing(root, res.StrictSingleLine > 0, res.StrictWouldRefuse > 0, authoring.PricedUnchecked)
			}
		} else {
			authoring.Record(root, authoring.RefusedApply)
		}
	default:
		if res.Applied && !res.DryRun {
			authoring.RecordRecent(root, res.Advisories)
		}
		switch {
		case res.Failed > 0:
			authoring.Record(root, authoring.RefusedApply)
		case res.DryRun:
			// ADR-079: a clean dry run landed nothing.
		default:
			authoring.Record(root, authoring.Applied)
		}
	}
	// ADR-200: every call that left anything on disk is counted here, once,
	// before a check runs and by no other phase: a clean commit, a commit whose
	// ledger update failed, and a partial commit alike.
	if MutationOf(res) != None {
		authoring.RecordNumbering(root, res.StaleNumbered, 0) // ADR-203: no landing has an unshown candidate
	}
	return l, nil
}

// newerLedger replaces each not-read refusal's reason with ADR-150's sentence
// when the ledger was written by a newer mrw (ADR-157 Decision 1): "has not been
// read" after a read looks like a bug, and its advice, to read the file or pass
// --force, cannot license the write. mrw_write has no stderr, so the refusal
// carries the cause; its kind stays NotRead, so stats still counts it.
func newerLedger(root string, res *apply.Result) {
	if res.Failed == 0 {
		return
	}
	writer, newer, err := seen.Newer(root)
	if err != nil || !newer {
		return
	}
	for i, h := range res.Hunks {
		switch h.Kind {
		case refusal.NotRead:
			res.Hunks[i].Reason = h.Path + " cannot be edited by this mrw: " + strings.TrimPrefix(seen.NewerNotice(writer), "mrw: ")
		case refusal.SourceNotRead:
			// ADR-167: the source is what could not be read, not the target.
			src, _, _, _ := h.Source()
			res.Hunks[i].Reason = src + " cannot be copied by this mrw: " + strings.TrimPrefix(seen.NewerNotice(writer), "mrw: ")
		}
	}
}

// NewerUntold is ADR-150's sentence when the ledger was written by a newer mrw
// and no refusal in res carries it, and "" otherwise (ADR-157 Decision 2): the
// CLI says it once for a write, in a refusal or after the landing, never both.
func NewerUntold(root string, res apply.Result) string {
	writer, newer, err := seen.Newer(root)
	if err != nil || !newer {
		return ""
	}
	for _, h := range res.Hunks {
		if h.Kind == refusal.NotRead || h.Kind == refusal.SourceNotRead {
			return ""
		}
	}
	return seen.NewerNotice(writer)
}

// CheckDue says whether this write's check runs: on a real landing, and not
// under CheckOff (ADR-205: whatever the files are).
func (l *Landed) CheckDue() bool {
	return l.Res.Applied && l.Res.Failed == 0 && l.p.req.Check != CheckOff
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
		written := CheckPaths(res.Files)
		// ADR-112: the baseline is what the write left on disk, taken now.
		before := Before(root, res)
		var opts []check.Option
		if removedCode(res.Files) {
			opts = append(opts, check.Whole())
		}
		cr, err := check.Run(ctx, root, cfg, written, opts...)
		if err != nil {
			authoring.Reclassify(root, authoring.Applied, authoring.CheckNotRun)
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
		authoring.Reclassify(root, authoring.Applied, authoring.CheckNotRun)
	case v.Check != nil && !v.Check.OK():
		authoring.Reclassify(root, authoring.Applied, authoring.FailedCheck)
	}
	if s, _, ok := StepStop(v.Then); ok && res.Applied && (v.Check == nil || v.Check.OK()) {
		if s.Ran {
			authoring.Reclassify(root, authoring.Applied, authoring.FailedCheck)
		} else {
			authoring.Reclassify(root, authoring.Applied, authoring.CheckNotRun)
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
		authoring.RecordPricing(root, res.StrictSingleLine > 0, res.StrictWouldRefuse > 0, outcome)
	}
}

// CheckPaths is the check's working set after a write. Unlinked and rename
// sources are gone, so confine cannot Stat them; their parent directory still
// exists and is what the check can honour. A rename dest is a Written file of
// its own. Whether the write touched code no longer decides whether a check is
// due (ADR-205), so the paths are all it answers.
func CheckPaths(files []apply.FileResult) (paths []string) {
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
			continue
		}
		if f.Written {
			add(f.Path)
		}
	}
	return paths
}

// removedCode says whether the write took a Go or module file away, by unlink
// or as the source of a rename. The check only ever sees its parent directory
// (CheckPaths), and the packages that import it are not among the paths, so a
// {packages}-only scoped_check runs the whole check instead (ADR-164
// Decision 4).
func removedCode(files []apply.FileResult) bool {
	for _, f := range files {
		if !f.Removed {
			continue
		}
		switch filepath.Base(f.Path) {
		case "go.mod", "go.sum", "go.work", "go.work.sum":
			return true
		}
		if filepath.Ext(f.Path) == ".go" {
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
