# ADR-101 Tasks

Implementation tasks for ADR-101: a check that did not run has no exit code of zero. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | a check that did not run reports `exit_code` -1; contract §195 | pending | — | `docs/adr/ADR-101-a-check-that-did-not-run-has-no-exit-code-of-zero/tasks/T1-no-exit-code-without-a-run.md` fence |
| T1 | a check that did not run reports `exit_code` -1; contract §195 | done | — | `docs/adr/ADR-101-a-check-that-did-not-run-has-no-exit-code-of-zero/tasks/T1-no-exit-code-without-a-run.md` fence |
Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| none | — | — | one task |

## Notes

- Engine go/no-go: ADR-101 owns `internal/check`; `internal/read`, `internal/apply`, `internal/plan`, `internal/seen`, `internal/state`, `internal/lines`, `internal/iter` and `internal/rooted` stay byte-identical.
- Contract section §195: §194 is the highest (ADR-100), found with the `sort -k2,2n` recipe on 2026-09-30.
