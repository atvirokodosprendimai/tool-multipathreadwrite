# ADR-094 Tasks

Implementation tasks for ADR-094: a step says what it checked. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T3 | T1, T2 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | A placeholder in a step is refused before anything is written | done | — | `docs/adr/ADR-094-a-step-says-what-it-checked/tasks/T1-a-placeholder-in-a-step-is-refused.md` fence |
| T2 | A passing step shows its last line | done | — | `docs/adr/ADR-094-a-step-says-what-it-checked/tasks/T2-a-passing-step-shows-its-last-line.md` fence |
| T3 | Every surface teaches what a step checked | done | — | `docs/adr/ADR-094-a-step-says-what-it-checked/tasks/T3-teach-what-a-step-checked.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | `check.Placeholder`, the placeholder refusal | T3 | T1 before T3 |
| T2 | the passing-step line, the `--quiet` usage | T3 | T2 before T3 |

T2 consumes nothing of T1's; it follows T1 because both edit `cmd/mrw/main.go` and
`scripts/contract.sh`.

## Notes

- `internal/check` changes in T1 only; `internal/read`, `apply`, `plan`, `seen` and `state` stay
  byte-identical in every task. The fences compare against the branch's merge-base with
  `origin/main` and refuse an untracked file there, as ADR-093's and ADR-096's do, so a sibling record
  merged first (ADR-096 changes `internal/read`) does not fail them; T2 and T3 also refuse any change in
  `internal/check` outside T1's two files (an edit inside `check.go` by T2 or T3 is left to review).
- Contract sections §178–§180 were allocated to this record; §178 (T1) and §179 (T2) are used, §180 is
  not.
