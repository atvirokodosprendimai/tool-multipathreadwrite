// Package writer lands a plan as the one writer on its checkout (ADR-075).
//
// Two mrw processes on one checkout — two CLI writes, or a CLI write beside an
// MCP server — each validated a plan against the file it had read and then
// renamed its result into place, so a later rename threw an earlier edit away
// while both printed "applied", exit 0: 45–53% of racing writers in the
// v1.25.1 adversarial round. A landing (applyCounted, through Land) holds a per-checkout lock from validation
// to the ledger update, so a writer whose file changed while it waited is
// refused by apply's own sha check instead.
package writer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"

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

// String is the name a receipt carries for m: none, partial or complete (#735).
func (m Mutation) String() string {
	switch m {
	case Partial:
		return "partial"
	case Complete:
		return "complete"
	}
	return "none"
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

// SyncError is a failure to make the tree stable AFTER the plan landed (ADR-210):
// every file is where the plan put it, the ledger has recorded it, and a caller
// reports the landed receipt beside it, as for a LedgerError. Ledger is the
// ledger's failure too, when it also failed, so neither error is lost.
type SyncError struct {
	Err    error // the *apply.SyncError
	Ledger error
}

func (e *SyncError) Error() string {
	s := e.Err.Error() + "; the files are written but not known to be on stable storage: do not send the plan again"
	if e.Ledger != nil {
		s += "; and the ledger could not record what landed: " + e.Ledger.Error()
	}
	return s
}

// Unwrap returns the sync's own error and the ledger's.
func (e *SyncError) Unwrap() []error {
	if e.Ledger != nil {
		return []error{e.Err, e.Ledger}
	}
	return []error{e.Err}
}

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
	res, err := applyFn(root, in, opt)
	m := MutationOf(res)
	if m == None {
		return res, 0, err
	}
	gen := bumpWrites(root)
	wrote, gone := ledgerChanges(root, res, opt)
	// Drop before Record: an unlink of c and a rename onto c in one plan leave
	// two records for c, and the one that holds is the file on disk.
	if lerr := seen.Drop(root, gone); lerr != nil {
		return res, gen, ledgerFailed(m, err, lerr)
	}
	if lerr := seen.Record(root, wrote); lerr != nil {
		return res, gen, ledgerFailed(m, err, lerr)
	}
	return res, gen, syncLanded(err)
}

// applyFn is apply.Apply, a seam for the tests of what a partial commit does to the
// ledger, the tally, the pricing and the recent ring: since ADR-209 no filesystem
// here makes a commit fail after a rename and leave a file written, so the tests hand
// the writer the Result such a commit would give.
var applyFn = apply.Apply

// ledgerChanges is what a write that changed the tree says to the ledger: the
// observations of the files whose path holds what mrw wrote, and the paths whose
// records are dropped. A file whose path an undo left missing or holding another
// process's file (NotHeld, ADR-209) is recorded as neither written nor known.
func ledgerChanges(root string, res apply.Result, opt apply.Options) (map[string]seen.Observation, []string) {
	wrote := map[string]seen.Observation{}
	var gone []string
	for _, f := range res.Files {
		if f.Removed {
			gone = append(gone, f.Path)
			continue
		}
		if f.NotProduced {
			// #712: a moved link's destination licenses nothing, and a record
			// an earlier read left at that path, under this spelling or another
			// that names the same directory entry, is dropped with it (Codex on
			// #725: it could otherwise license an edit through the link).
			gone = append(gone, f.Path)
			gone = append(gone, entryAliases(root, f.Path, opt.Seen)...)
			continue
		}
		if f.Written && f.NotHeld {
			continue
		}
		if f.Written && f.QuotedOnly {
			// ADR-190: a quote licensed these lines and the caller read nothing
			// else, so the file is known only where it was written. ADR-208
			// Decision 1a: those lines are licensed and not shown, as a plan's
			// and a --quiet read's are, so a later multi-line replace needs the
			// line after them read before it leans on them.
			wrote[f.Path] = seen.Observation{SHA: f.SHAAfter, Written: true, Spans: append([][2]int{}, f.Spans...)}
		} else if f.Written {
			wrote[f.Path] = seen.Observation{SHA: f.SHAAfter, Written: true}
		}
	}
	// Issue #597: the ledger is keyed by spelling, so the same file read as
	// A.TXT (or through an in-root symlink) keeps an entry at the sha it had
	// before this write. Left there, a write through that spelling is refused
	// as changed since mrw saw it. Dropped, its next lookup misses and finds
	// this write's record by os.SameFile, as apply's alias lookup already does.
	gone = append(gone, otherSpellings(root, res.Files, opt.Seen)...)
	return wrote, gone
}

// entryAliases returns the ledger keys, other than p, that name the same
// directory entry as p: os.SameFile on Lstat, so a directory alias reaches
// the moved link itself and not the file it points at (#712).
func entryAliases(root, p string, ledger map[string]apply.Seen) []string {
	fi, err := os.Lstat(filepath.Join(root, filepath.FromSlash(p)))
	if err != nil {
		return nil
	}
	var out []string
	for key := range ledger {
		if key == p || key == filepath.ToSlash(p) {
			continue
		}
		if ki, err := os.Lstat(filepath.Join(root, filepath.FromSlash(key))); err == nil && os.SameFile(fi, ki) {
			out = append(out, key)
		}
	}
	return out
}

// otherSpellings finds the ledger keys that still name a written file after
// landing (#597). A symlink anywhere in the checkout can point to the file.
func otherSpellings(root string, files []apply.FileResult, ledger map[string]apply.Seen) []string {
	named := map[string]bool{}
	var written []os.FileInfo
	for _, f := range files {
		named[f.Path] = true
		if !f.Written || f.Removed || f.NotProduced { // a moved link's target is not followed (#712)
			continue
		}
		if fi, err := os.Stat(filepath.Join(root, filepath.FromSlash(f.Path))); err == nil {
			written = append(written, fi)
		}
	}
	if len(written) == 0 {
		return nil
	}
	if len(ledger) < 128 {
		var aliases []string
		for key := range ledger {
			if !named[key] && namesWrittenFile(root, key, written) {
				aliases = append(aliases, key)
			}
		}
		return aliases
	}

	// A large ledger keeps the write lock held through many independent stats.
	// Bound the filesystem concurrency and collect all matches before returning.
	keys := make([]string, 0, len(ledger))
	for key := range ledger {
		if !named[key] {
			keys = append(keys, key)
		}
	}
	var matches [4][]string
	var wg sync.WaitGroup
	for i := range matches {
		batch := keys[i*len(keys)/len(matches) : (i+1)*len(keys)/len(matches)]
		wg.Go(func() {
			for _, key := range batch {
				if namesWrittenFile(root, key, written) {
					matches[i] = append(matches[i], key)
				}
			}
		})
	}
	wg.Wait()
	return slices.Concat(matches[:]...)
}

func namesWrittenFile(root, key string, written []os.FileInfo) bool {
	fi, err := os.Stat(filepath.Join(root, filepath.FromSlash(key)))
	if err != nil {
		return false
	}
	for _, w := range written {
		if os.SameFile(w, fi) {
			return true
		}
	}
	return false
}

// ledgerFailed is what applyCounted returns when the ledger could not record what
// landed. After a complete commit it is a LedgerError, which callers read as
// "the plan landed". After a partial one the commit error stays the error — a
// LedgerError there would be counted as a clean landing — and the ledger's
// failure is added to it.
func ledgerFailed(m Mutation, commitErr, lerr error) error {
	var perr *apply.SyncError
	if m == Complete && errors.As(commitErr, &perr) {
		return &SyncError{Err: perr, Ledger: lerr} // both errors kept
	}
	if m == Complete {
		return &LedgerError{Err: lerr}
	}
	return fmt.Errorf("%w; and the ledger could not record what landed: %w", commitErr, lerr)
}

// syncLanded turns apply's sync failure after landing into the writer's class,
// which a caller reads as "the plan landed". Any other error passes through.
func syncLanded(err error) error {
	var perr *apply.SyncError
	if errors.As(err, &perr) {
		return &SyncError{Err: perr}
	}
	return err
}
