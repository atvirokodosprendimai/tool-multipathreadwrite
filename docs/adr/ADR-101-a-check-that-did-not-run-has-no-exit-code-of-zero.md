# ADR-101: A check that did not run has no exit code of zero

**Status:** Accepted
**Accepted:** 2026-09-30 by Zy — asked whether to fix the `exit_code` 0 beside `"ran": false` that ADR-100 deferred, Zy answered "fix this exit code, do not leave this for dangling". The record's text was drafted after that answer and not shown to Zy before execution: this line is the session's reading of it, stated so it can be checked
**Date:** 2026-09-30
**Owner:** Zy
**Spec:** None — no spec stage
**Cross-references:** ADR-003, ADR-054, ADR-072, ADR-080, ADR-092, ADR-100
**Governs:** `internal/check/check.go`, `scripts/contract.sh`, `docs/adr/BACKLOG.md`, `README.md`, `AGENTS.md`
**Enforced-by:** `internal/check/noexit101_test.go::TestARunWithNoProcessHasExitCodeMinusOne`
**Invalidates:** none — checked. ADR-003's rule that `Ran` separates "no evidence" from "evidence of success" is kept and extended to `exit_code`; ADR-080 and ADR-092 already report -1 for a check or step with no exit status (could not start, timed out, interrupted, not run), and this record makes the three remaining paths agree. The ADR-100 BACKLOG entry is closed.
**Served-path change:** in `mrw check --json`'s receipt and in the `check` block of `mrw write --json`'s receipt, a check that did not run now reports `"exit_code": -1` where it reported `0`, in three cases: no check declared and no `go.mod`, a refused scope, and a check log that could not be created. Exit codes, the human report (which prints `check SKIPPED:` with no number) and every other field are unchanged.

## Context

**What was observed** (2026-09-30, `internal/check/check.go` at `e9620ea`, probed through `bin/mrw`):

1. `mrw check --json --full` in a tree with no harness and no `go.mod` prints
   `{"ran": false, "declared": false, "skipped": "no check declared and no go.mod found", "exit_code": 0}` and
   exits 2. `mrw write --check --json` there prints the same `check` block beside an applied write, exit 2.
   A consumer that reads `exit_code` alone reads a pass. Deferred by ADR-100 to BACKLOG; Zy asked for it now.
2. `Result.ExitCode` has no `omitempty`, and the zero value is 0. `run` sets -1 on every path where the
   process has no exit status — it never started (`:348`), it timed out (`:370`), it was interrupted (`:376`) —
   and steps not run carry -1 (`check.go:809`, `cmd/mrw/main.go:1746`). Three paths return before any of
   that and leave 0: the refused scope (`Run`, `:232`), no command (`Run`, `:236`), and the log that could not
   be created (`run`, `:309`).

**Audit of the class.** The class is *a `Result` or `StepResult` whose `exit_code` is 0 though no process
exited*. Enumerated 2026-09-30 by `mrw read --grep 'ExitCode' internal/check/ cmd/mrw/main.go internal/mcp/
--exclude '*_test.go'` and reading every `return` in `Run` and `run` and every `StepResult` literal:
- `Result`: **3** paths leave 0 (above) — all in scope. Every other path sets `Ran` with a real status or -1.
- `StepResult`: **0** — `RunSteps` starts each at -1 (`:809`) and copies the check's verdict only when the
  step ran; `runSteps` in `cmd/mrw` starts each not_run step at -1.
- Consumers: `check --json` (`checkReceipt`) and `write --json` (`receipt.Check`) serialize `Result`;
  `mrw_write` runs no check; `stats` reads `Ran` and `OK()`, never the number. The human report prints no
  number for a skipped check (`cmd/mrw/main.go:2126`).

## Existing Primitives Audit

- **-1 as "no exit status"** — reused; it is already the value for could-not-start, timed-out, interrupted
  and not-run.
- **`Result.OK()`** (`Ran && ExitCode == 0`) — unchanged, and unaffected: every changed path has `Ran` false.

## Decision

1. Every `check.Result` whose check produced no exit status carries `ExitCode` -1: `Run`'s refused scope and
   no-command returns, and `run` from its first line, so a return before the process exists says -1. A
   process that exited sets its own status, as before.
2. The receipts are unchanged in shape: `exit_code` stays present on every receipt.

## Alternatives Considered

- **Omit `exit_code` when the check did not run** — rejected: a consumer that decodes into an integer reads
  a missing field as 0, the same false pass; and it changes a shape ADR-054 and ADR-092 receipts pin.
- **Leave it, and teach "read `ran` first"** — what ADR-100 did; rejected by Zy on 2026-09-30.
- **A new sentinel (null, or a string)** — rejected: -1 is already the value every other no-status path uses.

## Component / Boundary Impact

`internal/check` only, plus prose and the contract. This record owns the `internal/check` change; every other
engine package stays byte-identical; `go.mod` keeps one requirement.

## Wiring & Contract Changes

| Surface | Change | Producer | Consumer(s) |
|---------|--------|----------|-------------|
| `check --json` receipt | `exit_code` -1 when the check did not run | T1 | CLI callers |
| `write --json` receipt `check` block | the same | T1 | CLI callers |
| `scripts/contract.sh` | §195 | T1 | CI Linux |
| README, AGENTS | say -1 when no check process exited | T1 | readers, the `mrw` skill |
| `docs/adr/BACKLOG.md` | the ADR-100 `exit_code` entry closed | T1 | readers |

## Inter-task Contracts

| Contract | Producing task | Consuming task(s) | Breaking? |
|----------|----------------|-------------------|-----------|
| none | — | — | one task |

## Implementation

See `tasks/README.md`: T1.

## Consequences

- **Positive:** no receipt says `exit_code` 0 about a check that did not exit 0.
- **Negative:** a consumer that compared `exit_code == 0` to find "nothing ran" now sees -1; it should have
  read `ran`.
- **Neutral:** exit codes and the human report are unchanged.

## Out of Scope

- The human report (permanent: boundary: it prints `check SKIPPED:` with no number when nothing ran, `cmd/mrw/main.go:2126`)
- `write --json` refusals before the plan is named, and `stats --json` (deferred: `docs/adr/BACKLOG.md` "From ADR-100")

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| a consumer used `exit_code == 0 && !ran` to mean "no check declared" | Low | Low | `skipped` names the reason; release notes say so |
| a path added later returns before `run` sets a status | Low | Med | `run` starts at -1, so an early return inherits it; the test drives the log-creation return |

## Rollback

Revert T1. No state or exit code changes.

## Follow-ups

- [x] Replace `**Enforced-by:** None — …` with `internal/check/noexit101_test.go::TestARunWithNoProcessHasExitCodeMinusOne` in T1's commit.
