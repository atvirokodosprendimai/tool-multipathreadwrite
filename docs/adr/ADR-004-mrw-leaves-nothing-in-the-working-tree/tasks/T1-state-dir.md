# Task ADR-004-T1: A per-checkout state directory resolves outside the working tree

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** M
**Produces:** `state.Dir` (T1), `state.Migrate` (T1)
**Consumes:** none
**Data dependency:** hermetic

## Goal

`internal/state` resolves `$XDG_STATE_HOME/mrw/<sha256(abs root)[:16]>/` for any
root, creates it on demand, records which checkout it belongs to, and copies a
legacy in-tree `.mrw/` across once without deleting anything.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/state/state.go` | add | `Dir`, `Migrate`, and the key derivation |
| `internal/state/state_test.go` | add | Resolution, XDG honoured, stability, migration, and the no-writes-under-root property |
| `cmd/mrw/main.go` | edit | Calls `state.Migrate` once per run and prints the notice — the call site that SELECTS the migration; without it a legacy ledger is silently abandoned |

## Ordered Steps

1. Write the failing test first (TDD red): `TestNoStateIsWrittenUnderTheRepoRoot`
   runs a read and a write against a temp root with `XDG_STATE_HOME` pointed at
   another temp dir, and asserts the root contains exactly the files the plan
   wrote. It must be red before `internal/state` exists.
2. Derive the base: `$XDG_STATE_HOME` when set and absolute, else
   `$HOME/.local/state`. Reject a relative `XDG_STATE_HOME` — the spec says it
   must be absolute, and honouring a relative one would put state back in the
   tree, which is the bug.
3. Key on `sha256` of the ABSOLUTE, symlink-resolved root, truncated to 16 hex
   characters. Absolute so two paths to one checkout agree; truncated because a
   64-character directory name is unreadable and 64 bits is ample here.
4. Write a `root` file containing the absolute path, so an orphan is
   identifiable rather than an anonymous hash.
5. `Migrate` copies `.mrw/seen` and `.mrw/iteration` into the state dir ONLY
   when the destination does not already hold that file, and reports what it
   copied. It never deletes.

## Acceptance

```bash
set -o pipefail
go test ./internal/state/ -v 2>&1 | tee /tmp/adr004t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr004t1.out \
  && go test ./internal/state/
```

## Tests

| Test name | File | Verifies | Covers |
|-----------|------|----------|--------|
| `TestDirIsUnderXDGStateHome` | `internal/state/state_test.go` | The resolved dir is under `$XDG_STATE_HOME/mrw`, never under the root | — |
| `TestDirFallsBackToLocalState` | `internal/state/state_test.go` | Unset `XDG_STATE_HOME` gives `~/.local/state/mrw` | — |
| `TestRelativeXDGStateHomeIsRejected` | `internal/state/state_test.go` | A relative `XDG_STATE_HOME` is refused rather than honoured — honouring it would put state back in the tree | — |
| `TestDirIsStableAndPerCheckout` | `internal/state/state_test.go` | Same root twice gives the same dir; two roots give different ones | — |
| `TestDirRecordsItsRoot` | `internal/state/state_test.go` | The `root` file names the absolute checkout path | — |
| `TestMigrateCopiesLegacyStateWithoutDeleting` | `internal/state/state_test.go` | A legacy `.mrw/seen` is copied and the original is still there | — |
| `TestMigrateNeverOverwritesNewerState` | `internal/state/state_test.go` | An existing state file is not clobbered by a legacy one | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestDirIsUnderXDGStateHome` |
| 2 — something selects it | `cmd/mrw/main.go` calls `state.Migrate` before every command; deleting that call makes `TestMigrateCopiesLegacyStateWithoutDeleting`'s CLI-level counterpart in T2 fail |
| 3 — the caller can discover it | `mrw seen` prints the resolved directory (T2); the migration notice names it on stderr |
| 4 — it is used | Every mrw invocation resolves it; nothing counts them |

## Mutation Log

- 2026-08-31 · 71b9018* · mutant killed · exit 1 · `internal/state/state.go` · Accepts a relative XDG_STATE_HOME instead of refusing it. Resolving one against the working directory is how state lands back inside a checkout — the bug this package exists to prevent. · acceptance-sha256:83d30152bcada43e76d98b169fa0c2ff4347a1102d95aeeab457051d7393cc43

## Invariants

- `Dir` never returns a path under the root it was given.
- The key is derived from the absolute, symlink-resolved root, so two spellings
  of one checkout share state.
- `Migrate` never deletes and never overwrites an existing destination file.

## Risks

- A relative `XDG_STATE_HOME` in someone's environment would silently relocate
  state; rejected explicitly rather than normalised, because normalising it
  guesses at intent.

## Stop Condition

Stop if a platform needs a different convention (Windows `%LOCALAPPDATA%`) —
that is a second location policy and belongs in its own record rather than in a
branch here.

## Out of Scope

- Moving the ledger and working set onto this — that is T2.
- Pruning orphaned state directories (deferred: docs/adr/BACKLOG.md)
- Windows conventions (deferred: docs/adr/BACKLOG.md)

## Verification Log
- 2026-08-31 · 71b9018* · exit 0 · `set -o pipefail …` · acceptance-sha256:83d30152bcada43e76d98b169fa0c2ff4347a1102d95aeeab457051d7393cc43
