# ADR-117 Tasks

Implementation tasks for ADR-117: mrw_read takes max_lines, stat and files_from. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | mrw_read takes max_lines, stat and files_from | done | none | `docs/adr/ADR-117-mrw-read-takes-max-lines-stat-and-files-from/tasks/T1-read-args.md` fence |
| T2 | a spec list drops a leading byte order mark and says so | done | T1 | `docs/adr/ADR-117-mrw-read-takes-max-lines-stat-and-files-from/tasks/T2-a-list-drops-a-leading-byte-order-mark.md` fence |
| T3 | a spec list drops every leading byte order mark and counts them | done | T2 | `docs/adr/ADR-117-mrw-read-takes-max-lines-stat-and-files-from/tasks/T3-a-list-drops-every-leading-byte-order-mark.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-117 owns no engine package.
- Contract sections §217, §341 (T2) and §342 (T3).
