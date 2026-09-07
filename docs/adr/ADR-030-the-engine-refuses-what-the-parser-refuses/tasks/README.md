# ADR-030 Tasks

Implementation tasks for ADR-030: The engine refuses what the parser refuses. See the parent ADR for
the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | The boundary refuses the enumerated list | done | — | `go test ./internal/apply/ -run 'TestTheEngineRefusesEveryShapeTheParserRefuses' …` and `go test ./internal/adversarial/ -run 'TestTheEngineAndTheParserRefuseInTheSameWords' …`, both in the fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | the enumerated engine-boundary refusals | — | Single task; nothing consumes it inside this record |

## Notes

- ⚠ **No contract row, deliberately.** `scripts/contract.sh` drives the built binary, which reaches `Apply` only through `plan.Parse`. A row would prove the parser's refusal and credit it to the engine — the vacuous-gate shape this corpus has now recorded five times. The Go test is the whole of rung 4, as it was for ADR-027-T3.
- The engine go/no-go applies with one exception this record owns: `internal/apply` changes here.
- ⚠ The list came from a BRANCH-BY-BRANCH walk of `plan.validate`, not from memory and not from probing the shapes that came to mind. The first cut did the latter, found seven, and called it complete; the review of PR #130 walked the branches and found two more — an insertion with a pattern RANGE and a `create` with a pattern address. Ten branches, not the two that ADR-026 and ADR-027 found one at a time, and not the seven the first pass reported.
