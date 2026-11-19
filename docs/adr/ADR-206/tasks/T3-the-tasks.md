# Task ADR-206-T3: 48 tasks in four classes

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** L
**Owner:** unassigned
**Produces:** `bench/stage-9/classes.json`; the tasks it names under `bench/tasks/`
**Consumes:** the corpus rules of `docs/bench/README.md`; repositories in `bench/repos.json`
**Data dependency:** needs the local clones `bench/repos.json` names
**Proof map:** v1
**Rests-on:** `each class has 12 tasks`, `every task is exact and creates no file`, `every oracle fails the base, passes the fix and refuses a stray edit`, `a follow-on task's first change moves the second's target`, `a control task's first change moves nothing`

## Goal

ADR-206 Decision 8 and the backlog entry's four classes: 12 single-site, 12 batch, 12 follow-on, 12 control, each task `exact: true`, none creating a file, none naming a tool. Existing tasks are used where they fit (the nine exact `multi-site` tasks for batch); the rest are written. For a follow-on task the reference patch has two hunks in one file, the first changing the line count above the second, and the instruction makes the second depend on the first's result. For a control task the first hunk keeps the line count.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `bench/tasks/s9-*/` | new | the tasks not already in the corpus |
| `bench/stage-9/classes.json` | new | class to task ids |
| `internal/bench/stage9_tasks_test.go` | new | the class rules |

## Ordered Steps

1. [S1] `TestStage9ClassesHoldTheirRules`, RED: no list. [proof: acceptance]
2. [S2] The tasks and the list; the corpus test `TestEveryOracleFailsOnBasePassesOnFixAndRefusesAStrayEdit` covers each. Mutants on the fixtures, each killed: a class of 11; a task that is not exact; a follow-on patch whose first hunk keeps the line count; a control patch whose first hunk does not. [proof: mutation]

## Acceptance

```bash
set -o pipefail
go test -count=1 -v ./internal/bench -run '^(TestStage9ClassesHoldTheirRules|TestEveryOracleFailsOnBasePassesOnFixAndRefusesAStrayEdit)$' > "${TMPDIR:-/tmp}/adr206-t3.out" \
  && [ "$(grep -c '^--- PASS: ' "${TMPDIR:-/tmp}/adr206-t3.out")" -eq 2 ] \
  && [ "$(jq '[.[] | length] | add' bench/stage-9/classes.json)" -eq 48 ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/apply internal/seen internal/check internal/state cmd/mrw ':(exclude)*_test.go'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestStage9ClassesHoldTheirRules` | `internal/bench/stage9_tasks_test.go` | 12 per class, each task loads, is exact, creates no file, and its reference patch has the shape its class requires | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test and the corpus test |
| 2 — something selects it | `--task` ids taken from `classes.json` |
| 3 — the caller can discover it | `docs/bench/README.md` |
| 4 — it is used | T5's smoke and T6 |

## Mutation Log

## Verification Log

## Invariants

- No existing task's instruction, oracle or fix changes.
- No task names a private project, customer or organization.

## Risks

- Twelve honest follow-on tasks are the hard part: each must read as one natural request.

## Stop Condition

The fence exits 0 and every gate in `AGENTS.md` passes.

## Out of Scope

- Tasks for a powered stage (deferred: docs/adr/BACKLOG.md)
