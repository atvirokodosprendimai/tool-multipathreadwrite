# ADR-107 Tasks

Implementation tasks for ADR-107: the last reads are bounded, and the recheck sees the leaf. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | none |
| 3 | T3 | none |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | a file over the edit limit is refused before it is read | pending | — | `docs/adr/ADR-107-the-last-reads-are-bounded-and-the-recheck-sees-the-leaf/tasks/T1-load-limit.md` fence |
| T2 | acknowledgement hashes in bounded memory | done | — | `docs/adr/ADR-107-the-last-reads-are-bounded-and-the-recheck-sees-the-leaf/tasks/T2-streamed-hash.md` fence |
| T3 | the recheck sees the leaf, and a failed ID load refuses | done | — | `docs/adr/ADR-107-the-last-reads-are-bounded-and-the-recheck-sees-the-leaf/tasks/T3-leaf-and-id.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| none | — | — | independent |

## Notes

- Engine go/no-go: ADR-107 owns `internal/apply`; every other engine package stays byte-identical.
