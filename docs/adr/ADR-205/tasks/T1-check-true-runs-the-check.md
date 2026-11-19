# Task ADR-205-T1: check: true runs the check

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `mrw_write` selecting `CheckDemand`; `CheckAuto` removed; the tool and schema descriptions; contract §352; the sentences in `AGENTS.md` and `README.md`
**Consumes:** `writer.CheckDemand`, the receipt's `check` object
**Data dependency:** none — hermetic
**Proof map:** v1
**Rests-on:** `check: true runs the check after a prose-only write`, `a check that could not run is an error with the applied receipt`, `without check: true no check runs`, `a failed check is still not an error`, `a due check is refused at depth 8 before the write`, `steps wait for the check`, `the zero mode is off`, `the markers exist`

## Goal

ADR-205 Decisions 1 to 5. Every row is a `mrw_write` call over the server; each fixture file is recorded in the read ledger as wholly read (the test's `licensed` helper seeds the ledger directly, with no `mrw_read` or ack call), and every row starts from a fresh tree, so a marker another row left cannot satisfy it. In a tree whose `.quality-harness.json` declares a check that appends a line to a marker file, and a step `after` that fails unless the marker exists:

| Call | Outcome |
|------|---------|
| a plan editing only `notes.md`, `check: true` | the edit landed; one marker line; `check.ran` true, `exit_code` 0; not `isError` |
| the same kind of plan, no `check` | the edit landed; no marker; no `check` key |
| `grep` with `sub` over `notes.md`, `check: true` | one marker line |
| a plan editing `a.go`, `check: true`, the check made to exit 1 | the edit landed; `check.ran` true, `exit_code` 1; not `isError` |
| a prose plan, `check: true`, `dry_run: true` | bytes unchanged; no marker; no `check` key |
| a prose plan, `check: true`, `then: ["after"]` | the step passed, which it can only after the check |
| a prose plan, `check: true`, `then: ["after"]`, the check made to exit 1 | the edit landed; the step is reported not run |
| a prose plan, `check: true`, with `MRW_STEP_DEPTH=8` | `isError` naming the depth; bytes unchanged |
| the same at depth 8, no `check` | the edit landed |

In a tree with no check declared and none inferable: a prose plan with `check: true` lands, and the answer is `isError` with `applied` true, `check.ran` false, `exit_code` -1 and a `skipped` reason.

And a `writer.Request` that sets no mode runs no check.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/tools.go` | edit | `CheckDemand` |
| `internal/writer/flow.go` | edit | `CheckAuto` and its branch removed |
| `internal/mcp/mcp.go`, `internal/mcp/schema.go` | edit | the descriptions |
| `internal/mcp/check205_test.go` | new | the Goal's table |
| `internal/mcp/check113_test.go`, `depth187_test.go`, `thread121_test.go`, `says098_test.go`, `era_test.go` and its golden | edit | they assert the prose skip or quote the old description |
| `internal/writer/flow113_test.go`, `writes127_test.go`, `partial164_test.go`, `stale200_test.go` | edit | they relied on the zero value being auto |
| `cmd/mrw/then092t5_test.go` | edit | a comment names the removed mode |
| `internal/mcp/instructions.go`, `cmd/opencode/mrw-plugin/src/index.ts` | edit | they state the old rule |
| `internal/writer/mode205_test.go` | new | the zero value |
| `scripts/contract.sh` | edit | §352 |
| `AGENTS.md`, `README.md` | edit | the sentences on `mrw_write`'s check |

## Ordered Steps

1. [S1] `TestMrwWriteCheckTrueRunsTheCheckOnAnyPlan`, RED on rows 1, 3, 6, 7 and 8 and on the commandless case; rows 2, 4, 5 and 9 hold today and stay. [proof: acceptance]
2. [S2] The mode and the removal. Mutants, each killed: `check: true` selects no check; a prose-only write skips it; a tree with no command returns no `check`; a failed check is `isError`; a check that could not run is not; a dry run runs it; the depth gate lets a due check through; steps run after a check that failed; the plan-maker path selects `CheckOff`; `CheckDemand` is made the zero value. The `format` and `ast_grep` sources reach the same `Prepare` call as the two the Goal drives, and are not driven separately. [proof: mutation]
3. [S3] Contract §352 drives `mrw mcp --enable` over stdio, each case in its own fresh tree: a prose-only write with `check: true` leaves the marker, paired with the same without it leaving none. The descriptions, the initialize instructions and the documents. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test -count=1 -v ./internal/mcp ./internal/writer -run '^(TestMrwWriteCheckTrueRunsTheCheckOnAnyPlan|TestARequestThatSetsNoModeRunsNoCheck)$' > "${TMPDIR:-/tmp}/adr205-fence.out" \
  && [ "$(grep -c '^--- PASS: ' "${TMPDIR:-/tmp}/adr205-fence.out")" -eq 2 ] \
  && grep -q '^# 352\. ' scripts/contract.sh \
  && ! grep -qw 'CheckAuto' internal/writer/flow.go internal/mcp/tools.go \
  && ! grep -rq 'touches code' internal/mcp/mcp.go internal/mcp/schema.go internal/mcp/instructions.go cmd/opencode/mrw-plugin/src/index.ts \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/apply internal/seen internal/check internal/state ':(exclude)*_test.go'
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestMrwWriteCheckTrueRunsTheCheckOnAnyPlan` | `internal/mcp/check205_test.go` | every row of the Goal's tables, by the marker file, the bytes on disk and the receipt | none | S1, S2 |
| `TestARequestThatSetsNoModeRunsNoCheck` | `internal/writer/mode205_test.go` | the zero value of the mode is off | none | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test and §352 |
| 2 — something selects it | `mrw_write`'s `check` argument |
| 3 — the caller can discover it | the tool's description and `check`'s schema description, which a host reads; `AGENTS.md`, `README.md` |
| 4 — it is used | telemetry is refused by ADR-009; the follow-up in the record |

## Mutation Log
- 2026-11-19 · 69de5b37* · mutant killed · exit 1 · `internal/mcp/tools.go` · check: true asks for no check · acceptance-sha256:75274f5e7767bed1adbcdf39c9e601d8263461ed9c5379c089386eb642cde7ff · covers:check: true runs the check after a prose-only write
- 2026-11-19 · 69de5b37* · mutant killed · exit 1 · `internal/mcp/tools.go` · a call without check runs one · acceptance-sha256:75274f5e7767bed1adbcdf39c9e601d8263461ed9c5379c089386eb642cde7ff · covers:without check: true no check runs
- 2026-11-19 · 69de5b37* · mutant killed · exit 1 · `internal/mcp/tools.go` · a check that could not run is not an error · acceptance-sha256:75274f5e7767bed1adbcdf39c9e601d8263461ed9c5379c089386eb642cde7ff · covers:a check that could not run is an error with the applied receipt
- 2026-11-19 · 69de5b37* · mutant killed · exit 1 · `internal/mcp/tools.go` · a failed check is an error · acceptance-sha256:75274f5e7767bed1adbcdf39c9e601d8263461ed9c5379c089386eb642cde7ff · covers:a failed check is still not an error
- 2026-11-19 · 69de5b37* · mutant killed · exit 1 · `internal/writer/flow.go` · the depth gate lets a due check through · acceptance-sha256:75274f5e7767bed1adbcdf39c9e601d8263461ed9c5379c089386eb642cde7ff · covers:a due check is refused at depth 8 before the write
- 2026-11-19 · 69de5b37* · mutant killed · exit 1 · `internal/writer/flow.go` · off is not the zero mode · acceptance-sha256:75274f5e7767bed1adbcdf39c9e601d8263461ed9c5379c089386eb642cde7ff · covers:the zero mode is off
- 2026-11-19 · 69de5b37* · mutant killed · exit 1 · `internal/writer/flow.go` · steps run after a check that did not pass · acceptance-sha256:75274f5e7767bed1adbcdf39c9e601d8263461ed9c5379c089386eb642cde7ff · covers:steps wait for the check

## Verification Log
- 2026-11-19 · 69de5b37* · exit 1 · `set -o pipefail …` · acceptance-sha256:75274f5e7767bed1adbcdf39c9e601d8263461ed9c5379c089386eb642cde7ff · ms:1209 · test-lock-sha256:d0d548351bcd96e566a8129833e2e2169eb22ba5e28862532cc640ddbe1ffb29 · test-lock-file:docs/adr/locks/d0d548351bcd96e566a8129833e2e2169eb22ba5e28862532cc640ddbe1ffb29.lock
  ```
  ```
- 2026-11-19 · 69de5b37* · exit 0 · `set -o pipefail …` · acceptance-sha256:75274f5e7767bed1adbcdf39c9e601d8263461ed9c5379c089386eb642cde7ff · ms:675
- 2026-11-19 · 69de5b37* · exit 0 · `set -o pipefail …` · acceptance-sha256:75274f5e7767bed1adbcdf39c9e601d8263461ed9c5379c089386eb642cde7ff · ms:1570
- 2026-11-19 · 69de5b37* · exit 0 · `set -o pipefail …` · acceptance-sha256:75274f5e7767bed1adbcdf39c9e601d8263461ed9c5379c089386eb642cde7ff · ms:1412
- 2026-11-19 · 69de5b37* · exit 0 · `set -o pipefail …` · acceptance-sha256:75274f5e7767bed1adbcdf39c9e601d8263461ed9c5379c089386eb642cde7ff · ms:1409
- 2026-11-19 · 69de5b37* · exit 0 · `set -o pipefail …` · acceptance-sha256:75274f5e7767bed1adbcdf39c9e601d8263461ed9c5379c089386eb642cde7ff · ms:1604
- 2026-11-19 · 69de5b37* · exit 0 · `set -o pipefail …` · acceptance-sha256:75274f5e7767bed1adbcdf39c9e601d8263461ed9c5379c089386eb642cde7ff · ms:1637
- 2026-11-19 · 69de5b37* · exit 0 · `set -o pipefail …` · acceptance-sha256:75274f5e7767bed1adbcdf39c9e601d8263461ed9c5379c089386eb642cde7ff · ms:2780
- 2026-11-19 · 69de5b37* · exit 0 · `set -o pipefail …` · acceptance-sha256:75274f5e7767bed1adbcdf39c9e601d8263461ed9c5379c089386eb642cde7ff · ms:1903
- 2026-11-19 · afabe090* · exit 0 · `set -o pipefail …` · acceptance-sha256:75274f5e7767bed1adbcdf39c9e601d8263461ed9c5379c089386eb642cde7ff · ms:1303

## Invariants

- `mrw write` on the CLI behaves exactly as before.
- No receipt key is added, removed or renamed.

## Risks

- The fence's `# 352.` clause is a marker; the contract gate proves the section.

## Stop Condition

The fence exits 0, `adr-debt` reports nothing broken or unreceipted, every gate in `AGENTS.md` passes with CI's Python suites and `scripts/measure.sh`, and the record is Accepted by Zy.

## Out of Scope

- Making a failed check `isError`, or a check that could not run not one (permanent: boundary: ADR-113 Decision 4 stands as it is)
