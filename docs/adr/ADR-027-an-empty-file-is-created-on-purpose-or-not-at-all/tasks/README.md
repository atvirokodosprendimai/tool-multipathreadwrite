# ADR-027 Tasks

Implementation tasks for ADR-027: An empty file is created on purpose, or not at all. See the parent
ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | A body-less create is refused, and body=0 is the deliberate empty file | done | — | `go test ./internal/plan/ -run 'TestACreateWithNoBodyIsRefusedUnlessItSaysBodyZero' …` |
| T2 | The contract drives both shapes, and the docs say which is which | done | — | `grep -q '^# 65\. ' scripts/contract.sh && ./scripts/contract.sh` |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | the body-less `create` refusal and its wording | T2 | T1 before T2 — the row drives the built binary, so the refusal must exist before the row can be red for the right reason |

## Notes

- The engine go/no-go applies and is not lifted: `internal/read`, `internal/apply`, `internal/seen`, `internal/check` and `internal/state` stay byte-identical against the merge base. This record owns `internal/plan` only.
- ⚠ The refusal must leave no file behind. A `create` that is refused and still creates the file would be worse than the behaviour this record removes, so §65 asserts the file's absence rather than only the exit code.
