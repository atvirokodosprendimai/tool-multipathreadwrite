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
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/seen"
)

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

// Apply applies in under root's write lock and records what landed before it
// releases the lock: a written file as wholly known (ADR-002, ADR-005), an
// unlinked or renamed-away one dropped.
//
// opt.Seen must be the ledger the caller loaded BEFORE calling. Loaded under
// the lock, a writer that waited would validate against the previous writer's
// whole-file licence, and its line numbers, counted in an older read, would
// land on the new content with exit 0. Loaded before, apply's sha check sees
// that the file changed and refuses. The check is the caller's, run after
// Apply returns: a five-minute check must not hold every other writer.
func Apply(root string, in []apply.Input, opt apply.Options) (apply.Result, error) {
	release, err := seen.LockWrites(root)
	if err != nil {
		return apply.Result{DryRun: opt.DryRun}, err
	}
	defer release()
	if inside != nil {
		inside()
	}
	res, err := apply.Apply(root, in, opt)
	if err != nil || !res.Applied || res.DryRun {
		return res, err
	}
	wrote := map[string]seen.Observation{}
	var gone []string
	for _, f := range res.Files {
		if f.Removed {
			gone = append(gone, f.Path)
			continue
		}
		if f.Written {
			wrote[f.Path] = seen.Observation{SHA: f.SHAAfter}
		}
	}
	if err := seen.Drop(root, gone); err != nil {
		return res, &LedgerError{Err: err}
	}
	if err := seen.Record(root, wrote); err != nil {
		return res, &LedgerError{Err: err}
	}
	return res, nil
}
