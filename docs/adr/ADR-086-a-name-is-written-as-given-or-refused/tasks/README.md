# ADR-086 Tasks

Implementation tasks for ADR-086: a name is written as given, or refused before anything is written. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Waves

| Wave | Tasks | Depends-on |
|------|-------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 2 | T3, T4 | T1 |

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T3 | T1 |
| 4 | T4 | T1 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|-------------|
| T1 | A plan header keeps every byte | done | — | `docs/adr/ADR-086-a-name-is-written-as-given-or-refused/tasks/T1-a-header-keeps-every-byte.md` fence |
| T2 | A name the filesystem refuses is refused at staging | done | — | `docs/adr/ADR-086-a-name-is-written-as-given-or-refused/tasks/T2-a-refused-name-writes-nothing.md` fence |
| T3 | Windows refuses a component it would read as U+FFFD | done | — | `docs/adr/ADR-086-a-name-is-written-as-given-or-refused/tasks/T3-windows-refuses-a-name-it-would-rewrite.md` fence |
| T4 | A --json plan refuses a name its receipt cannot represent | done | — | `docs/adr/ADR-086-a-name-is-written-as-given-or-refused/tasks/T4-json-refuses-a-name-it-cannot-represent.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-086 owns `internal/plan` (T1) and `internal/apply` (T2); T3 changes `internal/rooted` and T4 `cmd/mrw`, neither locked. `internal/read`, `internal/seen`, `internal/check`, `internal/state`, `internal/lines`, `internal/iter` and `internal/subproc` stay byte-identical.
