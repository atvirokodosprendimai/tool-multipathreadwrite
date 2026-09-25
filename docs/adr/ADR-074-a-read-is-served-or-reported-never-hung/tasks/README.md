# ADR-074 Tasks

Implementation tasks for ADR-074: a read is served or reported, never hung. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Waves

| Wave | Tasks | Depends-on |
|------|-------|------------|
| 1 | T1, T2, T3, T4 | none |

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | none |
| 3 | T3 | none |
| 4 | T4 | none |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|-------------|
| T1 | ast-grep runs through subproc, and an interrupt stops it; contract §147 | done | — | `docs/adr/ADR-074-a-read-is-served-or-reported-never-hung/tasks/T1-ast-grep-runs-through-subproc.md` fence |
| T2 | A named FIFO, socket or device is reported, not opened; contract §148 | done | — | `docs/adr/ADR-074-a-read-is-served-or-reported-never-hung/tasks/T2-a-named-fifo-is-reported.md` fence |
| T3 | The MCP page is sized by its encoded length; contract §149 | done | — | `docs/adr/ADR-074-a-read-is-served-or-reported-never-hung/tasks/T3-the-mcp-page-is-sized-by-its-encoded-length.md` fence |
| T4 | `--files-from` names a long line; the MSYS hint names both variables | done | — | `docs/adr/ADR-074-a-read-is-served-or-reported-never-hung/tasks/T4-files-from-and-msys-name-what-they-mean.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-074 owns `internal/read/read.go`, `walk.go` and `astgrep.go`, `internal/mcp/tools.go`, `internal/subproc/subproc.go` and `internal/check/check.go`. `internal/apply`, `internal/plan`, `internal/seen`, `internal/state`, `internal/lines`, `internal/iter`, `internal/rooted` and the rest of `internal/read`, `internal/check` and `internal/subproc` stay byte-identical.
