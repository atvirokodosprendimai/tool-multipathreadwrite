# ADR-100 Tasks

Implementation tasks for ADR-100: every `mrw check --json` refusal is a document. See the parent ADR.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | every `mrw check --json` refusal is one document; contract §193 | done | — | `docs/adr/ADR-100-every-check-json-refusal-is-a-document/tasks/T1-every-refusal-is-a-document.md` fence |
| T2 | the depth refusal answers first; contract §194 | done | — | `docs/adr/ADR-100-every-check-json-refusal-is-a-document/tasks/T2-the-depth-limit-answers-first.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | `refuse` defined before every refusal (T1) | T2 | T2 moves the depth call to the top of the Action, inside T1's shape |

## Notes

- Engine go/no-go: ADR-100 owns no engine package; every one stays byte-identical against the merge-base.
- Contract sections §193 and §194: §192 is the highest (ADR-099), found with the `sort -k2,2n` recipe on 2026-09-30.
