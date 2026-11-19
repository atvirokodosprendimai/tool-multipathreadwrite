# Task ADR-206-T3: 48 tasks in four classes

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** L
**Owner:** unassigned
**Produces:** `bench/stage-9/classes.json`; the tasks it names under `bench/tasks/`; the corpus test's registered shapes and counts
**Consumes:** the corpus rules of `docs/bench/README.md`; repositories in `bench/repos.json`
**Data dependency:** needs the local clones `bench/repos.json` names, or the oracle checks skip, which the fence forbids
**Proof map:** v1
**Rests-on:** `each class has 12 distinct tasks`, `every task is exact, creates no file and its require holds on the reference`, `every one of the 48 oracle checks ran`, `a class's patch has its class's shape`

## Goal

ADR-206 Decision 8. A *site* is one change region of the reference patch as `LoadTask` parses it. The classes:

| Class | Rule on the reference patch |
|-------|------------------------------|
| single-site | one site |
| batch | five or more sites in three or more files |
| follow-on | two sites in one file; the first changes the file's line count |
| control | two sites in one file; the first keeps the line count |

48 distinct ids, 12 a class, each `exact: true`, none creating a file, each with a `require` the reference result satisfies, none naming a tool or a private project. An existing task is used only if it meets its class's rule; the rest are written, with the shapes `follow-on` and `control` and the new counts registered in the corpus test. Whether a follow-on instruction reads as one natural request, its second change depending on the first, is a person's judgement, recorded per task in `bench/stage-9/tasks.md`.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `bench/tasks/s9-*/` | new | the tasks not already in the corpus |
| `bench/stage-9/classes.json`, `bench/stage-9/tasks.md` | new | the list; the per-task note |
| `internal/bench/task.go`, `internal/bench/oracle_test.go` | edit | the two shapes; the corpus test's counts |
| `internal/bench/stage9_tasks_test.go` | new | the class rules |

## Ordered Steps

1. [S1] `TestStage9ClassesHoldTheirRules`, RED: no list. [proof: acceptance]
2. [S2] The tasks and the list. Mutants on a copy of the list and its fixtures, each killed: a class of 11; a repeated id; a task that is not exact; a task whose `require` the reference misses; a follow-on patch whose first site keeps the line count; a control patch whose first does not; a batch of four sites. [proof: mutation]
3. [S3] The per-task notes. [proof: human: each follow-on and control instruction read against its patch]

## Acceptance

```bash
set -o pipefail
go test -count=1 -v ./internal/bench -run '^(TestStage9ClassesHoldTheirRules|TestEveryOracleFailsOnBasePassesOnFixAndRefusesAStrayEdit)$' > "${TMPDIR:-/tmp}/adr206-t3.out" \
  && [ "$(grep -c '^--- PASS: Test' "${TMPDIR:-/tmp}/adr206-t3.out")" -eq 2 ] \
  && ! grep -q 'SKIP' "${TMPDIR:-/tmp}/adr206-t3.out" \
  && [ "$(jq '[.[] | .[]] | unique | length' bench/stage-9/classes.json)" -eq 48 ] \
  && [ "$(jq '[.[] | length] | unique' -c bench/stage-9/classes.json)" = '[12]' ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/apply internal/seen internal/check internal/state internal/writer cmd/mrw ':(exclude)*_test.go'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestStage9ClassesHoldTheirRules` | `internal/bench/stage9_tasks_test.go` | 48 distinct ids, 12 a class; each loads, is exact, creates no file, its `require` holds on the reference result, and its patch has its class's shape | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test and the corpus test |
| 2 — something selects it | `--task` ids taken from `classes.json` |
| 3 — the caller can discover it | `docs/bench/README.md`, `bench/stage-9/tasks.md` |
| 4 — it is used | T5's smoke and T6 |

## Mutation Log

## Verification Log

## Invariants

- No existing task's instruction, oracle or fix changes.
- No task names a private project, customer or organization.

## Risks

- Twelve honest follow-on tasks are the hard part, and no test can judge them.

## Stop Condition

The fence exits 0 with no skip and every gate in `AGENTS.md` passes.

## Out of Scope

- Tasks for a powered stage (deferred: docs/adr/BACKLOG.md)
