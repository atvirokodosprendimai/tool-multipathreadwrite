# ADR-066 Tasks

Implementation tasks for ADR-066: a rename is checked before anything is written, a failed path-op commit is undone as a unit, and a partial commit says what it wrote. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | none |
| 3 | T3 | T1, T2 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|-------------|
| T1 | rename destinations: `Lstat` errors refused at validation; directories made while staging; contract §118; campaign probe | done | — | `docs/adr/ADR-066-a-plan-that-cannot-commit-whole-says-what-it-wrote/tasks/T1-rename-directories-are-staged.md` fence |
| T2 | a failed path-op commit undoes every unlink and rename | done | — | `docs/adr/ADR-066-a-plan-that-cannot-commit-whole-says-what-it-wrote/tasks/T2-a-failed-path-op-commit-is-undone-as-a-unit.md` fence |
| T3 | commit-failure receipts: ok / failed / skipped by what reached disk; `PARTIALLY APPLIED`; contract §119 | done | — | `docs/adr/ADR-066-a-plan-that-cannot-commit-whole-says-what-it-wrote/tasks/T3-a-partial-commit-says-what-it-wrote.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|----------------|
| T1 | `abortStage` shared by the staging loops | T3 | T3 builds the commit verdicts beside it |
| T2 | `commitRenameFn` seam and path-op undo | T3 | T3's receipts describe what T2 leaves on disk |

## Notes

- No `§NN` row in any Tests table (ADR-052/054 lesson).
- Engine go/no-go: ADR-066 owns `internal/apply` and `cmd/mrw/main.go`'s `report`. `internal/read`, `internal/plan`, `internal/seen`, `internal/check`, `internal/state` and `internal/mcp` stay byte-identical, and every fence asserts it.
- Branch order: ADR-066 merges to `main` before ADR-065's branch starts, so neither record's fences see the other's diff.
