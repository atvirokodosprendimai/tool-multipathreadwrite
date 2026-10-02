# ADR-118 Tasks

Implementation tasks for ADR-118: a pattern address picks the Nth match. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | occurrence=N picks the Nth start match, every earlier match served | done | none | `docs/adr/ADR-118-a-pattern-address-picks-the-nth-match/tasks/T1-occurrence.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-118 owns `internal/plan` and `internal/apply`.
- Contract section §218.
