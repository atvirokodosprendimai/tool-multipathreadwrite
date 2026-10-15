# Task ADR-160-T2: the Stage 4 corpus, drawn by the registered rules

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** L (machine time: a corpus build)
**Owner:** Zy
**Produces:** `bench/tasks/<wide-rename and check-coupled ids>/`, `bench/stage-4-tasks.txt`, `bench/swe-bench/selection-adr-160.json`
**Consumes:** `bench mine`, `bench swebench-select --salt --min-hunks --exclude --dataset`, the shapes `wide-rename` and `check-coupled` (T1)
**Data dependency:** needs agentsmemory's whole history (the clone is shallow on 2026-10-15) and the SWE-bench Verified file at revision `78f471bf655a3137b2e8a75af1501690ec009ec3`
**Proof map:** v1
**Rests-on:** `the category counts`, `every new task's oracle ran`, `the task list`, `the slice file`

## Goal

10 `wide-rename` and 8 `check-coupled` tasks with oracle and reference fix, the 24-id task list, and the 20-instance `swe-multi-hunk` selection, each drawn by T1's code in registered order.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `bench/tasks/<id>/task.json`, `reference.patch` | add | 18 own tasks |
| `bench/stage-4-tasks.txt` | add | the 10, the 8 and the six `single-line` control ids, one per line |
| `bench/swe-bench/selection-adr-160.json` | add | the slice: rule, dataset, revision, harness, the 20 ids, the unnamed-files subset |
| `bench/repos.json` | edit | each repository the slice draws that Stage 3 did not use |

## Ordered Steps

1. [S1] See the fence red: no task of either new shape exists. [proof: acceptance]
2. [S2] `git fetch --unshallow origin` in the agentsmemory clone; pin `origin/main`'s sha; record the branch, the sha and that the clone is no longer shallow. Run `bench mine --kind wide-rename` and `--kind check-coupled` at that sha and record each command and its full output. Fewer than 14 `wide-rename` or 12 `check-coupled` candidates stops the task. [proof: human: the executor records the fetch, the sha, the commands, the candidate counts and the order]
3. [S3] For each of the first 10 `wide-rename` candidates: Y of X's byte length, absent from the repository, written into `task.json`; the reference by whole-word replacement, comments included; `go build ./...`, `go vet ./...` and `gofmt -l` on the touched files pass, or the candidate is skipped for the next one and the skip recorded. [proof: mutation]
4. [S4] For each of the first 8 `check-coupled` candidates: the parent pinned; `go test` of the package passes at the parent in the tree bench builds, or the candidate is skipped and the skip recorded; the instruction from the template; the reference the commit's diff. [proof: mutation]
5. [S5] Write `bench/stage-4-tasks.txt`. Run `bench swebench-select --salt adr-160 --min-hunks 5 --exclude bench/swe-bench/selection.json --dataset <file> --n 20` into `selection-adr-160.json`; clone each repository it draws that Stage 3 did not use, add it to `bench/repos.json`, and check each `base_commit` is present. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
ids=$(grep -l '"shape": "\(wide-rename\|check-coupled\)"' bench/tasks/*/task.json | xargs -n1 dirname | xargs -n1 basename)
test "$(grep -l '"shape": "wide-rename"' bench/tasks/*/task.json | wc -l | tr -d ' ')" = 10 \
  && test "$(grep -l '"shape": "check-coupled"' bench/tasks/*/task.json | wc -l | tr -d ' ')" = 8 \
  && test "$(grep -c . bench/stage-4-tasks.txt)" = 24 \
  && for id in $ids; do grep -qx "$id" bench/stage-4-tasks.txt || exit 1; done \
  && jq -e '(.selected | length) == 20 and .rule != null and (.unnamed | type) == "array"' bench/swe-bench/selection-adr-160.json > /dev/null \
  && out=$(go test ./internal/bench/ -count=1 -v -run '^TestEveryOracleFailsOnBasePassesOnFixAndRefusesAStrayEdit$' 2>&1) \
  && for id in $ids; do grep -qE "^ +--- PASS: TestEveryOracleFailsOnBasePassesOnFixAndRefusesAStrayEdit/$id " <<<"$out" || exit 1; done
```

Each new task's own subtest must PASS: the oracle test skips a task whose clone or commit is unreachable (`internal/bench/oracle_test.go:50-51`) and passes while the fixtures run, so the test's own verdict cannot carry this.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestEveryOracleFailsOnBasePassesOnFixAndRefusesAStrayEdit` | `internal/bench/oracle_test.go` | every new task's oracle fails its base, passes its reference, refuses a stray line | none | S3, S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the task directories, the list and the selection file |
| 2 — something selects it | `bench run --task` from the list; `--selection` names the slice |
| 3 — the caller can discover it | `docs/bench/README.md` |
| 4 — it is used | T3 and T4 |

## Mutation Log
(empty until execute)

## Invariants

- No existing task changes; none is superseded by this task.
- A candidate is skipped only for a reference that does not build, vet or format, or a package red at its parent, never for anything an attempt did.

## Risks

- A same-length name reads oddly: it is a rename instruction, and both configurations get the same one.

## Stop Condition

Fewer candidates than the category needs; or a new task's subtest skips.

## Out of Scope

- Running anything (T3).

## Verification Log
(empty until execute)
