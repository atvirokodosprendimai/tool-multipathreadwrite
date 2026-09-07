# ADR-029 Tasks

Implementation tasks for ADR-029: One file is one observation, whatever the plan calls it. See the
parent ADR for the decision.

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
| T1 | The observation is resolved once and `covered()` consumes it | done | — | `go test ./internal/adversarial/ -run 'TestAnAliasSpellingIsTheSameFileToThePerLineLedger' …` |
| T2 | The contract drives it through the built binary | done | — | `grep -q '^# 67\. ' scripts/contract.sh && ./scripts/contract.sh` |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | the per-line check consuming the alias-resolved observation | T2 | T1 before T2 — the row drives the built binary, so the resolution must exist before the row can be red for the right reason |

## Notes

- The engine go/no-go applies with one exception this record owns: `internal/apply` changes here. `internal/read`, `internal/plan`, `internal/seen`, `internal/check` and `internal/state` stay byte-identical against the merge base.
- ⚠ Neither alias is portable, and each half is covered on exactly one CI operating system: the symlink on Linux, the case-only variant on Windows, where NTFS is case-insensitive. Probe at runtime and skip; never assert the platform.
- ⚠ The fixture must be TWO-SIDED. A test asserting only that the alias write is refused is green against a fix that refuses every alias, which is issue #47 undone.
