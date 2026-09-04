---
paths:
  - "**/*_test.go"
  - "scripts/contract.sh"
---

# Tests: a green test that never reached the defect is the failure mode here

## A fixture must reach the bug, not resemble the report

A regression test built from a bug report can pass without executing the broken path. Prove it by
mutation: break the fix and confirm the test goes red. When a mutant survives, the first hypothesis
is "my fixture never reaches that branch", not "the mutation was too weak" — instrument and see
which branch runs.

The measured case: `read.Run` emits a per-file header, so ~3,000 files push the REPORT over the cap
and the index branch fires first; the served-page bug lived in a band only ~1,500 files reaches. The
test was green and hollow at 3,000 AND at 12,001 before it could kill anything.

## Follow a continuation; do not inspect it

A cursor, `next_index`, or a next page is tested by FOLLOWING it to exhaustion and comparing the
reassembled union to the whole — with no overlap between pages and a bounded page count so a
non-terminating cursor fails instead of hanging. A presence check passed over a continuation that
nothing could follow (no `after` argument existed) and then over an off-by-one that yielded 7,999 of
8,000 — one lost at the page boundary.

## Assert the thing, not its shadow

- A ledger claim reads the ledger (or attempts the write it should refuse), never the response.
- A bound is measured in the unit it is declared in: §52 counted 4,073 runes against a 4,096-BYTE
  bound while Go counted 4,091 bytes.
- A test that names only the bad path can take a different branch from the one that is broken;
  pair it with a valid sibling.
- `t.Cleanup` inside a helper shared by several tests binds to the FIRST caller's `t`. A built
  binary belongs in `TestMain`.

## Names and the fence

Test names are sentences stating the property (`TestAFallbackRootThatIsNotAProjectIsRefused`); the
Acceptance fence greps `^--- PASS: Name\b`, so a name that is a prefix of another name is a hole.
`CONTRIBUTING.md` says why an exit code is never read through a pipe.
