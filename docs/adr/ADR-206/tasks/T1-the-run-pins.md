# Task ADR-206-T1: a run pins its mrw, its instructions and its spend

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** L
**Owner:** unassigned
**Produces:** `bench run` and `bench canary` `--mrw-sha256`, `--mrw-instructions`; `bench run` `--keep-state-file`, `--keep-diff`, `--max-spend`; the `run.json` and receipt keys; resume and verify checks; the spend stop's exit status
**Consumes:** `verifyAsset`, `stageQH`, `keepHookFiles`, `readsBack`
**Data dependency:** none — hermetic. The runner tests' fake agent is extended: it runs `mrw`, copies the prompt file it was given, and emits a chosen cost or none
**Proof map:** v1
**Rests-on:** `a wrong hash refuses the run`, `the attempt runs the pinned bytes`, `the instructions are the file's`, `the kept file is beside the receipt`, `the run stops on known spend`, `a missing cost is not zero`, `a receipt with another hash fails resume and verify`, `an altered kept file fails verify`, `the diff is kept`, `a canary with other hashes is refused`, `a run without the flags is unchanged`

## Goal

ADR-206 Decisions 2, 3, 5 and 6. The rows drive `Run` with the runner tests' fixture task (which `--task` cannot select); one more test case parses the real flag set and checks each flag reaches its option.

| Run | Outcome |
|-----|---------|
| `--mrw-sha256` naming another hash | refused before any attempt, the message naming both hashes |
| the right hash, and the file at `--mrw` replaced between two attempts | both attempts ran the hashed bytes (the fake records the hash of the `mrw` it ran); `run.json` and both receipts carry `mrw_sha256` |
| `--mrw-instructions FILE` | the prompt file the fake was given ends with the file's bytes; the binary's `instructions` was not run; `run.json` carries `mrw_instructions_sha256` |
| `--keep-state-file mrw/x.jsonl`, the fake mrw writing it under its state home | the file is beside the receipt, its hash in the receipt, the setting in `run.json`; when not written, no file and no key |
| `--max-spend 1.00`, two planned attempts costing 0.60 each | both run, a third planned attempt does not; the exit status is the spend stop's own; a second invocation with `--max-spend 2.00` is refused as another plan |
| `--max-spend 9`, three receipts with no cost | the run stops, naming the missing costs |
| a receipt edited to carry another `mrw_sha256` | a resume refuses it, and `bench verify` fails naming the attempt |
| a kept file altered or removed after the run | `bench verify` fails naming the attempt |
| `--keep-diff` | `<attempt id>.diff` beside the receipt holds the attempt's diff against its base, its hash in the receipt |
| configuration H with `--mrw-sha256` | the Read hook's `bin/mrw` is the pinned bytes |
| a canary recorded with another mrw or instruction hash | the run refuses it |
| none of the flags | `run.json` and the receipts hold what the same run holds today, plus `bench_revision` and `init_model` (compared without start and wall times) |

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/bench/runner.go`, `canary.go`, `cmd/bench/main.go` | edit | the flags, the keys, the stop, the checks |
| `internal/bench/runner_test.go` | edit | the fake agent's new abilities, off unless a test asks |
| `internal/bench/stage9_test.go` | new | the Goal's table |
| `docs/bench/README.md` | edit | the flags |

## Ordered Steps

1. [S1] `TestARunPinsItsMrwItsInstructionsAndItsSpend`, RED on every row but the last. [proof: acceptance]
2. [S2] The flags and checks. Mutants, each killed: the hash is not compared; the attempt links the live path; the binary's instructions are run; the kept file is not copied; the diff is taken against the wrong commit; verify skips a kept file; the stop sums nothing; a null cost counts as zero; resume skips validation; verify skips the hash; the canary's hashes are not compared. [proof: mutation]
3. [S3] `docs/bench/README.md`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test -count=1 -v ./internal/bench -run '^TestARunPinsItsMrwItsInstructionsAndItsSpend$' > "${TMPDIR:-/tmp}/adr206-t1.out" \
  && [ "$(grep -c '^--- PASS: TestARunPinsItsMrwItsInstructionsAndItsSpend ' "${TMPDIR:-/tmp}/adr206-t1.out")" -eq 1 ] \
  && grep -q -- '--max-spend' docs/bench/README.md && grep -q -- '--mrw-sha256' docs/bench/README.md \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/apply internal/seen internal/check internal/state internal/writer cmd/mrw ':(exclude)*_test.go'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestARunPinsItsMrwItsInstructionsAndItsSpend` | `internal/bench/stage9_test.go` | every row of the Goal's table, from `run.json`, the receipts, the exit status and the fake's own record of what it ran and was given | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | the flags of `bench run` and `bench canary` |
| 3 — the caller can discover it | `bench run --help`, `docs/bench/README.md` |
| 4 — it is used | T5's smoke and T6 |

## Mutation Log

## Verification Log

## Invariants

- A run that passes none of the flags behaves as before, with two recorded keys more.
- No package of `mrw` changes.

## Risks

- A mutant on this package has cost about four minutes to grade (an estimate from ADR-192's task): they run one at a time.

## Stop Condition

The fence exits 0, `adr-debt` reports nothing broken or unreceipted, and every gate in `AGENTS.md` passes.

## Out of Scope

- A stop across run directories in Go (permanent: boundary: six figures sum to the stage's stop, ADR-206 Decision 8)
