# ADR-120 Tasks

Implementation tasks for ADR-120: Windows owns its process tree. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | the job sequence, kernel32 behind it, and the Windows grandchild tests | done | none | `docs/adr/ADR-120-windows-owns-its-process-tree/tasks/T1-job.md` fence |
| T2 | the docs say a Windows child stops with mrw | done | T1 | `docs/adr/ADR-120-windows-owns-its-process-tree/tasks/T2-docs.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine go/no-go: ADR-120 owns no engine package; `internal/check` changes in a test file only.
- No contract section: see the record's Out of Scope.
- T1 S3, the Windows evidence: CI run 37057757674 on fd17179 (PR #325), every windows shard green;
  `internal/subproc` — the real grandchild tests, which have no skip — ok under `go test` and `-race`, and the two
  un-skipped timed-out-check tests ran in their shards (Git's sh is on the runners' PATH).
