# ADR-132 Tasks

Implementation tasks for ADR-132: a refusal exits and reads the same wherever it is found. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | a target-caused refusal before the first rename exits 1 | done | none | `docs/adr/ADR-132-a-refusal-exits-and-reads-the-same-wherever-it-is-found/tasks/T1-exit.md` fence |
| T2 | a receipt says it once, in the caller's words | done | T1 | `docs/adr/ADR-132-a-refusal-exits-and-reads-the-same-wherever-it-is-found/tasks/T2-words.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- T1 owns `internal/apply/apply.go`'s staging returns; T2 its read-advice words.
- Contract sections §231 (T1) and §232 (T2); T1 also moves §119 and §168.
