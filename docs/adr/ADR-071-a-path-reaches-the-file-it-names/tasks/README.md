# ADR-071 Tasks

Implementation tasks for ADR-071: a path reaches the file it names on disk, or is refused. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Waves

| Wave | Tasks | Depends-on |
|------|-------|------------|
| 1 | T1, T2 | none |
| 2 | T3 | T1 |
| 3 | T4 | T3 |
| 4 | T5 | T1, T2, T3 |

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | none |
| 3 | T3 | T1 |
| 4 | T4 | T3 |
| 5 | T5 | T1, T2, T3 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|-------------|
| T1 | A junction is followed like a symlink | done | — | `docs/adr/ADR-071-a-path-reaches-the-file-it-names/tasks/T1-a-junction-is-followed-like-a-symlink.md` fence |
| T2 | Two creates of one file are refused; contract §141 | done | — | `docs/adr/ADR-071-a-path-reaches-the-file-it-names/tasks/T2-two-creates-of-one-file-are-refused.md` fence |
| T3 | A Win32 alias is refused on Windows | done | — | `docs/adr/ADR-071-a-path-reaches-the-file-it-names/tasks/T3-a-win32-alias-is-refused.md` fence |
| T4 | The Windows suite reaches its branches | done | — | `docs/adr/ADR-071-a-path-reaches-the-file-it-names/tasks/T4-the-windows-suite-reaches-its-branches.md` fence |
| T5 | The prose says what the code does | done | — | `docs/adr/ADR-071-a-path-reaches-the-file-it-names/tasks/T5-the-prose-says-what-the-code-does.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-071 owns `internal/apply` (T2) and the `internal/check` test files (T4). `internal/read`, `internal/plan`, `internal/seen`, `internal/state`, `internal/lines`, `internal/iter` and `internal/check` outside its test files stay byte-identical.
