// Package writer lands a plan as the one writer on its checkout (ADR-075).
//
// Two mrw processes on one checkout — two CLI writes, or a CLI write beside an
// MCP server — each validated a plan against the file it had read and then
// renamed its result into place, so a later rename threw an earlier edit away
// while both printed "applied", exit 0: 45–53% of racing writers in the
// v1.25.1 adversarial round. Apply holds a per-checkout lock from validation
// to the ledger update, so a writer whose file changed while it waited is
// refused by apply's own sha check instead.
package writer

import (
	"fmt"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

// Mutation is how much of a plan reached the tree (ADR-102): the one answer
// every tally and receipt reads. Asking Applied alone called a commit that
// failed after a file had landed "nothing" (ADR-066).
type Mutation int

const (
	// None — nothing reached the tree: a dry run, a refusal, or a commit that
	// failed before its first file landed.
	None Mutation = iota
	// Partial — the commit failed after at least one file landed.
	Partial
	// Complete — the plan applied whole.
	Complete
)

// MutationOf reads res for how much of its plan reached the tree.
func MutationOf(res apply.Result) Mutation {
	switch {
	case res.DryRun:
		return None
	case res.Applied:
		return Complete
	}
	for _, f := range res.Files {
		if f.Written {
			return Partial
		}
	}
	return None
}

// inside runs once the lock is held, before the plan is applied. A test uses
// it to see that no two writers are ever inside at once.
var inside func()

// LedgerError is a failure to update the ledger AFTER the plan landed: the
// tree changed and mrw's record of it did not, so a caller reports the landed
// receipt beside it rather than as a plan that failed.
type LedgerError struct{ Err error }

func (e *LedgerError) Error() string { return e.Err.Error() }

// Unwrap returns the ledger's own error.
func (e *LedgerError) Unwrap() error { return e.Err }

// applyCounted applies in under root's write lock and records what landed
// before it releases the lock: a written file as wholly known (ADR-002,
// ADR-005), an unlinked or renamed-away one dropped. A commit that failed after
// some files landed records those too (ADR-102): mrw wrote them, so it knows
// them. It also returns the checkout's write counter as this write left it
// (ADR-127): bumped under the lock when anything landed, so a check that runs
// after the lock is released can tell how many other writes landed meanwhile;
// zero when nothing landed or the counter could not move.
//
// opt.Seen must be the ledger the caller loaded BEFORE calling. Loaded under
// the lock, a writer that waited would validate against the previous writer's
// whole-file licence, and its line numbers, counted in an older read, would
// land on the new content with exit 0. Loaded before, apply's sha check sees
// that the file changed and refuses. The check is the caller's, run after it
// returns: a five-minute check must not hold every other writer.
func applyCounted(root string, in []apply.Input, opt apply.Options) (apply.Result, int64, error) {
	release, err := seen.LockWrites(root)
	if err != nil {
		return apply.Result{DryRun: opt.DryRun}, 0, err
	}
	defer release()
	if inside != nil {
		inside()
	}
	res, err := apply.Apply(root, in, opt)
	m := MutationOf(res)
	if m == None {
		return res, 0, err
	}
	gen := bumpWrites(root)
	wrote := map[string]seen.Observation{}
	var gone []string
	for _, f := range res.Files {
		if f.Removed {
			gone = append(gone, f.Path)
			continue
		}
		if f.Written {
			wrote[f.Path] = seen.Observation{SHA: f.SHAAfter, Written: true}
		}
	}
	// Drop before Record: an unlink of c and a rename onto c in one plan leave
	// two records for c, and the one that holds is the file on disk.
	if lerr := seen.Drop(root, gone); lerr != nil {
		return res, gen, ledgerFailed(m, err, lerr)
	}
	if lerr := seen.Record(root, wrote); lerr != nil {
		return res, gen, ledgerFailed(m, err, lerr)
	}
	return res, gen, err
}

// ledgerFailed is what Apply returns when the ledger could not record what
// landed. After a complete commit it is a LedgerError, which callers read as
// "the plan landed". After a partial one the commit error stays the error — a
// LedgerError there would be counted as a clean landing — and the ledger's
// failure is added to it.
func ledgerFailed(m Mutation, commitErr, lerr error) error {
	if m == Complete {
		return &LedgerError{Err: lerr}
	}
	return fmt.Errorf("%w; and the ledger could not record what landed: %w", commitErr, lerr)
}
