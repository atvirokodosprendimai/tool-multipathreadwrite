# Task ADR-206-T1: a run pins its mrw, its instructions and its spend

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** unassigned
**Produces:** `bench run --mrw-sha256`, `--mrw-instructions`, `--keep-state-file`, `--max-spend`; their `run.json` and receipt keys
**Consumes:** `verifyAsset`, `stageQH`, `keepHookFiles` (existing)
**Data dependency:** none — hermetic (a fake agent and a fake mrw, as the runner's tests use)
**Proof map:** v1
**Rests-on:** `a wrong hash refuses the run`, `the attempt runs the pinned bytes`, `the instructions are the file's`, `the kept file is beside the receipt`, `the run stops on spend`, `a missing cost is not zero`, `a run without the flags is unchanged`

## Goal

ADR-206 Decisions 2, 3, 5 and 6. In a run driven by the runner tests' fake agent:

| Run | Outcome |
|-----|---------|
| `--mrw-sha256` naming another hash | refused before any attempt, the message naming both hashes |
| the right hash, and the file at `--mrw` replaced after the run starts | the attempt still runs the bytes that were hashed; `run.json` and the receipt carry `mrw_sha256` |
| `--mrw-instructions FILE` | the appended text is the file's bytes, the binary's `instructions` is not run, `run.json` carries `mrw_instructions_sha256` |
| `--keep-state-file NAME`, the fake mrw writing `NAME` in its state directory | the file is beside the receipt and its hash in the receipt; when not written, no file and no key |
| `--max-spend 1.00`, receipts summing to 1.20 | no further attempt starts; the run exits as the disk floor does and resumes when the figure is raised |
| `--max-spend`, three receipts whose cost is null | the run stops, naming the missing costs |
| none of the four flags | `run.json` and the receipts are byte for byte what the same run writes today |

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/bench/runner.go`, `cmd/bench/main.go` | edit | the flags, the plan and receipt keys, the stop |
| `internal/bench/stage9_test.go` | new | the Goal's table |
| `docs/bench/README.md` | edit | the four flags |

## Ordered Steps

1. [S1] `TestARunPinsItsMrwItsInstructionsAndItsSpend`, RED on every row but the last. [proof: acceptance]
2. [S2] The flags. Mutants, each killed: the hash is not compared; the attempt links the live path; the binary's instructions are appended beside the file's; the kept file is not copied; the stop sums nothing; a null cost counts as zero. [proof: mutation]
3. [S3] `docs/bench/README.md`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test -count=1 -v ./internal/bench -run '^TestARunPinsItsMrwItsInstructionsAndItsSpend$' > "${TMPDIR:-/tmp}/adr206-t1.out" \
  && [ "$(grep -c '^--- PASS: ' "${TMPDIR:-/tmp}/adr206-t1.out")" -eq 1 ] \
  && grep -q -- '--max-spend' docs/bench/README.md \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/apply internal/seen internal/check internal/state cmd/mrw ':(exclude)*_test.go'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestARunPinsItsMrwItsInstructionsAndItsSpend` | `internal/bench/stage9_test.go` | every row of the Goal's table, from `run.json`, the receipts and the fake's own record of what it was run with | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | the four flags of `bench run` |
| 3 — the caller can discover it | `bench run --help` and `docs/bench/README.md` |
| 4 — it is used | T5's smoke and T6 |

## Mutation Log

## Verification Log

## Invariants

- A run that passes none of the four flags writes what it wrote before.
- No package of `mrw` changes.

## Risks

- The runner's tests are slow (about four minutes a mutant on this package): the mutants run one at a time.

## Stop Condition

The fence exits 0, `adr-debt` reports nothing broken or unreceipted, and every gate in `AGENTS.md` passes.

## Out of Scope

- A stage-wide stop across run directories in Go (permanent: boundary: the script sums them, ADR-206 Decision 6)
