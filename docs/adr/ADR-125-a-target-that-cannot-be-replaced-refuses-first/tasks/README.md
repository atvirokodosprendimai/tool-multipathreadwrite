# ADR-125 Tasks

Implementation tasks for ADR-125: a target that cannot be replaced refuses before any rename. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | a target mrw cannot open is refused on its hunk, naming why | done | none | `docs/adr/ADR-125-a-target-that-cannot-be-replaced-refuses-first/tasks/T1-receipt.md` fence |
| T2 | on Windows a held target fails before any rename | pending | T1 | `docs/adr/ADR-125-a-target-that-cannot-be-replaced-refuses-first/tasks/T2-replace.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-125 owns `internal/apply`.
- Contract section §223.
