# ADR-157 Tasks

Implementation tasks for ADR-157: a notice says what happened, and only advice that can help. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Task Index

| ID | Title | Status | Depends-on | Acceptance |
|----|-------|--------|------------|------------|
| T1 | a newer ledger is named first, once, and before an ack | done | none | `docs/adr/ADR-157-a-notice-says-what-happened-and-what-helps/tasks/T1-the-newer-ledger-first.md` fence |
| T2 | the stale notice claims no discard, help is silent, seen names the ledger | done | T1 | `docs/adr/ADR-157-a-notice-says-what-happened-and-what-helps/tasks/T2-the-ledger-named-where-it-is-used.md` fence |
| T3 | the own-state refusal, a doubled BOM, and JSON-path quoting | done | none | `docs/adr/ADR-157-a-notice-says-what-happened-and-what-helps/tasks/T3-advice-that-fits.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Notes

- Engine packages `internal/seen`, `internal/read`, `internal/apply` and `internal/check` change; this record owns those changes.
- Contract sections §260 (T1), §261 (T2), §262 (T3).
- T1 edits a test that ADR-150 T1 locks (`cmd/mrw/writer150_test.go`) and contract §254, re-locks that task with `adr-verify <task> --relock --replace-hashes` after confirming the change is the one this record makes, and every record is swept with adr-lint before merging. T2's notice keeps the two phrases ADR-136's tests pin, so it re-locks nothing unless its S1 search finds otherwise.
