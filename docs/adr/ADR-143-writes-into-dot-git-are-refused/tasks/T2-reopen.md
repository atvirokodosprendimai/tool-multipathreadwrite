# Task ADR-143-T2: A path reopened by name is judged for `.git` again

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `rooted.HasGitComponent`, the check in `tree.rel`
**Consumes:** `inGit` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a directory swapped for a link to .git after validation does not carry a create, an unlink or a rename destination into .git`

## Goal

Decision 6 of the record, with a test that fails before it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/rooted/gitdir.go` | edit | `HasGitComponent` |
| `internal/apply/tree.go` | edit | `tree.rel` refuses a path with a `.git` component |
| `internal/apply/swap143_test.go` | add | the test, on ADR-106's swap seams |

## Ordered Steps

1. [S1] Write `TestAWriteThroughAParentSwappedForDotGitStaysOut` (a create, an unlink and a rename destination, each after `stageFileFn` swaps `sub` for a link to `.git`). Confirm RED.
2. [S2] `HasGitComponent` and its call in `tree.rel`. Mutant: the check removed. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/apply/ -count=1 -timeout 300s -run 'TestAWriteThroughAParentSwappedForDotGitStaysOut' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAWriteThroughAParentSwappedForDotGitStaysOut \(' "$out" \
  && go test ./internal/apply/ ./internal/rooted/ -count=1 -timeout 900s \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAWriteThroughAParentSwappedForDotGitStaysOut` | `internal/apply/swap143_test.go` | after a parent is swapped for a link to `.git`, a create, an unlink and a rename destination write nothing into `.git` and the plan is not applied | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `rooted.HasGitComponent` |
| 2 — something selects it | `tree.rel`, which every staging, rename and removal passes |
| 3 — the caller can discover it | the refusal names `.git` and git as the tool |
| 4 — it is used | reproduced in the test with ADR-106's seams; no telemetry (ADR-009) |

## Invariants

- ADR-106's confinement is unchanged: a path that leaves the root is refused as before.
- A name that is not `.git` is never refused by this check.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if `tree.rel` is reached by a path mrw must legitimately open inside a `.git` (it is not: validation refuses every write there first).

## Out of Scope

- The instant between the check and the syscall (permanent: boundary: the finest grain mrw has; see the record)

## Mutation Log

## Verification Log
