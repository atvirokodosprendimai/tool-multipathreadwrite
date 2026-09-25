# ADR-069 Tasks

Implementation tasks for ADR-069: a caller-supplied path reaches mrw exactly as typed, or is refused. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

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
| T1 | The CLI refuses a positional its parser trimmed; contract §128 | done | — | `docs/adr/ADR-069-a-caller-supplied-path-reaches-mrw-exactly/tasks/T1-refuse-a-padded-positional.md` fence |
| T2 | `--files-from` and the working set keep a line's trailing space; contract §129 | pending | — | `docs/adr/ADR-069-a-caller-supplied-path-reaches-mrw-exactly/tasks/T2-line-formats-keep-the-path.md` fence |
| T3 | A rename destination keeps its trailing space; contract §130 | pending | — | `docs/adr/ADR-069-a-caller-supplied-path-reaches-mrw-exactly/tasks/T3-rename-destination-keeps-the-path.md` fence |
| T4 | apply_patch and search_replace keep a path's trailing space; contract §131 | pending | — | `docs/adr/ADR-069-a-caller-supplied-path-reaches-mrw-exactly/tasks/T4-foreign-formats-keep-the-path.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- No `§NN` row in any Tests table.
- Engine go/no-go: ADR-069 owns `internal/iter` (T2) and `internal/apply` (T3). `internal/read`, `internal/plan`, `internal/seen`, `internal/check`, `internal/state` and `internal/lines` stay byte-identical.
