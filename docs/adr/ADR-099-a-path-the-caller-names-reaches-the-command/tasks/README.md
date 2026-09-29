# ADR-099 Tasks

Implementation tasks for ADR-099: a path the caller names reaches the command. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | none |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | `mrw check --full PATH` is refused; contract §191 | done | — | `docs/adr/ADR-099-a-path-the-caller-names-reaches-the-command/tasks/T1-full-takes-no-path.md` fence |
| T2 | `help` is a path to read, write and check; contract §192 | done | — | `docs/adr/ADR-099-a-path-the-caller-names-reaches-the-command/tasks/T2-help-is-a-path.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| none | — | — | independent |

## Notes

- Engine go/no-go: ADR-099 owns no engine package; every one stays byte-identical against the merge-base.
- Contract sections §191 and §192: §189 and §190 are ADR-098's (PR #284), so this record takes the next two.
