# Task ADR-009-T1: Record the outcome of every plan, in counts only

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** unassigned
**Produces:** `authoring.Outcome` (closed vocabulary), `authoring.Record(root string, o Outcome, cat Category) error`, `authoring.Load(root string) (Tally, error)`, `authoring.Tally`
**Consumes:** `state.Path` (ADR-004, existing)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the closed vocabulary`, `the fail-open read`, `the absence of caller content in the written file`

## Goal

`mrw write` records what happened to the plan it was handed — counts per outcome, and per parse-error
category when the plan did not parse — in a state-directory file that contains nothing of the
caller's work.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/authoring/authoring.go` | add | `Outcome`, `Category`, `Record`, `Load`, `Tally` — the vocabulary and the file format |
| `internal/authoring/authoring_test.go` | add | its tests, including the boundary test that is this ADR's `Enforced-by` |
| `cmd/mrw/main.go` | edit | **THE CALL SITE.** The `write` action classifies its own outcome and calls `authoring.Record`. Without this line the package is unreachable, which is the defect this repository keeps shipping — `rooted.Descendable` was deleted on 2026-09-03 for exactly that |
| `scripts/contract.sh` | edit | §36: a write leaves a tally behind, a refused plan is counted as refused, and the file names nothing of the caller's |

## Ordered Steps

1. [S1] Write the failing tests first (TDD red): a recorded outcome round-trips through `Load`; an
   unreadable tally yields an empty `Tally` and no error; the written file contains no substring of
   a plan body, path or anchor fed to `Record`.
2. [S2] Define `Outcome` and `Category` as **typed constants, not strings** — `applied`,
   `refused_parse`, `refused_guard`, `refused_unseen`, `refused_boundary`, `failed_check`; categories
   drawn from what `plan.Parse` already distinguishes. A closed type is what makes the boundary
   checkable rather than a promise.
3. [S3] Implement `Record` and `Load` over `state.Path(root, "authoring")`, following `internal/seen`'s
   shape: load whole, rewrite whole, one short line per counter.
4. [S4] Make the read FAIL OPEN — an unreadable or malformed tally is discarded and `Record` still
   returns nil. Measurement that can fail a write is worse than no measurement, and this is the rule
   that keeps it from becoming load-bearing. [proof: mutation]
5. [S5] Call it from `cmd/mrw`'s `write` action, mapping the outcome it already computes for the exit
   status onto an `Outcome`. One switch, beside the one that already exists. [proof: mutation]
6. [S6] Add contract §36 driving the real binary: a successful write increments `applied`, a plan
   with a bad header increments `refused_parse`, and `grep` over the tally file finds no fragment of
   the plan that produced it. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/authoring/ -run 'TestTally|TestTheTally' -v 2>&1 | tee /tmp/adr009-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr009-t1.out \
  && go test ./... \
  && grep -q '^# 36\.' scripts/contract.sh \
  && ./scripts/contract.sh
```

The `grep -q '^# 36\.'` clause is not decoration. §15 already existed when ADR-007-T3's fence named
it, so that fence passed on an untouched `contract.sh` from the day it was written — found
2026-09-03. Confirm 36 is unused before relying on this line.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTallyRoundTripsThroughLoad` | `internal/authoring/authoring_test.go` | a recorded outcome is readable back | — | S1, S3 |
| `TestTallyCountsEachOutcomeSeparately` | `internal/authoring/authoring_test.go` | the vocabulary is not collapsed | — | S1, S2 |
| `TestAnUnreadableTallyFailsOpen` | `internal/authoring/authoring_test.go` | garbage in the file yields an empty tally and no error | — | S1, S4 |
| `TestRecordNeverFailsAWrite` | `internal/authoring/authoring_test.go` | `Record` returns nil even when the state dir is unwritable | — | S4 |
| `TestTheTallyNeverRecordsPlanContentOrPaths` | `internal/authoring/authoring_test.go` | **the boundary.** Feed a plan body, a path and an anchor through `Record`; assert no substring of any of them appears in the written bytes | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the five tests above |
| 2 — something selects it | `cmd/mrw/main.go`'s `write` action (S5). Deleting that call makes contract §36's "a successful write increments applied" row red — that is the mutation, and it is inside the Acceptance fence rather than beside it |
| 3 — the caller can discover it | n/a: no declared interface until T2 adds `mrw stats` |
| 4 — it is used | the tally itself, once T3 publishes a reading — this is the rung this ADR exists to stop leaving empty |

## Mutation Log

## Invariants

- `Record` never returns an error that a caller would act on, and never fails a write.
- The tally file contains only counter names from the closed vocabulary and integers.
- `mrw write`'s exit statuses, output and ledger behaviour are unchanged; a caller who never runs
  `mrw stats` cannot tell this task landed.

## Risks

- The call site is one line and one line is easy to lose in a refactor. Mitigated by rung 2's
  mutation being inside the fence.
- Fail-open hides a broken tally forever. Accepted: a tally that can break a write is the worse
  failure, and T2's `mrw stats` shows an empty tally plainly enough to notice.

## Stop Condition

Stop if recording the outcome requires `cmd/mrw` to compute anything it does not already compute for
the exit status. The tally must be a projection of a decision already made, not a second opinion
about what happened — two answers to "did this apply?" is the defect class this project refuses.

## Out of Scope

- `mrw stats` — T2's job.
- Any content beyond counts and category names (permanent: boundary: stated in the parent ADR)

## Verification Log
