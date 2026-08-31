# Task ADR-004-T2: The ledger and the working set move to the state directory, and `mrw seen` shows where

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** L (cross-boundary)
**Owner:** M
**Produces:** `mrw seen` (T2)
**Consumes:** `state.Dir` (T1), `state.Migrate` (T1)
**Data dependency:** hermetic

## Goal

`internal/seen` and `internal/iter` store under the state directory instead of
`.mrw/`, a legacy in-tree file is picked up once, and `mrw seen` prints the
resolved location and the ledger.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/seen/seen.go` | edit | Path comes from `state.Dir(root)`, not `root + "/.mrw/seen"` |
| `internal/seen/seen_test.go` | edit | Tests point `XDG_STATE_HOME` at a temp dir |
| `internal/iter/iter.go` | edit | Same move for the working set |
| `internal/iter/iter_test.go` | edit | Same |
| `cmd/mrw/main.go` | edit | `seenCmd` registered in `Commands` — the registry line that makes `mrw seen` reachable |
| `scripts/contract.sh` | edit | A row asserting the working tree is untouched by a read |
| `README.md` | edit | Documents the location, `mrw seen`, `anchor=` being a raw substring, and a codegen step inside `check` |
| `.gitignore` | edit | `/.mrw/` kept with a comment saying it is legacy-only |

## Ordered Steps

1. Write the failing test first (TDD red): `internal/seen` and `internal/iter`
   round-trip tests with `XDG_STATE_HOME` set must fail while the packages
   still write to `.mrw/`.
2. Replace the `File` constants with a call to `state.Dir(root)`, keeping every
   exported signature — `Load(root)`, `Record(root, obs)`, `Forget(root, …)`,
   `Save(root, set)` — so no consumer changes shape.
3. Have `Load` fall back to the legacy in-tree file when the state dir holds
   none, so a caller who has not run a migrating command still sees their data.
4. Add `mrw seen`: print the resolved directory FIRST, then the ledger contents,
   so "where is it" is answered before "what is in it".
5. Add a contract row: after `mrw read --stat`, the working tree contains no
   `.mrw` directory.
6. Document in README: the state location, `mrw seen`, that `anchor=` is a plain
   substring with no escape processing, and that a codegen step belongs inside
   the declared `check` command (`templ generate && go test {packages}`).

## Acceptance

```bash
set -o pipefail
go test ./internal/seen/ ./internal/iter/ -v 2>&1 | tee /tmp/adr004t2.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr004t2.out \
  && go test ./... \
  && ./scripts/contract.sh
```

## Tests

| Test name | File | Verifies | Covers |
|-----------|------|----------|--------|
| `TestRoundTrip` | `internal/seen/seen_test.go` | Ledger round-trips through the state dir | — |
| `TestLedgerIsNotWrittenIntoTheRoot` | `internal/seen/seen_test.go` | After Record, the root holds no `.mrw` | — |
| `TestLegacyInTreeLedgerIsStillRead` | `internal/seen/seen_test.go` | A pre-existing `.mrw/seen` is honoured when the state dir has none | — |
| `TestRoundTrip` | `internal/iter/iter_test.go` | Working set round-trips through the state dir | — |
| `TestWorkingSetIsNotWrittenIntoTheRoot` | `internal/iter/iter_test.go` | After Save, the root holds no `.mrw` | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestLedgerIsNotWrittenIntoTheRoot` |
| 2 — something selects it | `mrw seen` is registered in `rootCommand`'s `Commands` slice; `scripts/contract.sh` asserts the tree stays clean after a read, which fails if either package reverts to `.mrw/` |
| 3 — the caller can discover it | `mrw seen` prints the directory; README documents it; the migration notice names it |
| 4 — it is used | Every read and write resolves it; nothing counts them |

## Mutation Log

- 2026-08-31 · 71b9018* · mutant inconclusive · exit 1 · `internal/seen/seen.go` · Sends WRITES back into the working tree. The read path would still work, so only a test asserting the tree stays clean can catch it — which is the property ADR-004 is about. · acceptance-sha256:36dba1b6f763cb16251cadaf1cfb648c74ed3b4e80b16829b46173e8ca1eb491
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-08-31 · 71b9018* · mutant killed · exit 1 · `internal/seen/seen.go` · Sends WRITES back into the working tree. Reads would still succeed, so only a test asserting the tree stays clean catches it — which is the whole property of ADR-004. · acceptance-sha256:36dba1b6f763cb16251cadaf1cfb648c74ed3b4e80b16829b46173e8ca1eb491

## Invariants

- No exported signature in `seen` or `iter` changes — the location is a detail
  neither package's callers hold.
- A legacy in-tree file is read when the state dir has none, and is never
  deleted.
- `mrw read --stat` creates nothing under the root.

## Risks

- A caller with a committed `.mrw/seen` keeps reading it until they delete it,
  which could mask the fix. Accepted: reading it is strictly better than
  ignoring data they have, and the migration notice tells them it is there.

## Stop Condition

Stop if honouring the legacy path turns out to be load-bearing for anyone — that
would mean in-tree state has a real constituency and the decision needs
revisiting rather than a compatibility shim growing.

## Out of Scope

- Deleting the legacy `.mrw/` (permanent: not this tool's business — see the
  parent ADR)
- Pruning orphaned state directories (deferred: docs/adr/BACKLOG.md)

## Verification Log
- 2026-08-31 · 71b9018* · exit 0 · `set -o pipefail …` · acceptance-sha256:36dba1b6f763cb16251cadaf1cfb648c74ed3b4e80b16829b46173e8ca1eb491
