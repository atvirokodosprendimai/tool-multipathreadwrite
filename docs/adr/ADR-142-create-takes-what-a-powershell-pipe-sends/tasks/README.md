# ADR-142 Tasks

Implementation tasks for ADR-142: `mrw write --create` takes what a PowerShell pipe sends. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | `--create` drops a pipe terminator, names a BOM, and says a path exists | done | none | `docs/adr/ADR-142-create-takes-what-a-powershell-pipe-sends/tasks/T1-pipe.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Touches `internal/ingest`, `internal/apply` and `cmd/mrw`.
- Contract section §243.
