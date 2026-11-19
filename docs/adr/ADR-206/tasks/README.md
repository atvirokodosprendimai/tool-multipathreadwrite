# ADR-206 Tasks

Implementation tasks for ADR-206: Stage 9 is run as three separate bench runs per model, with two builds of one tag that differ in one refusal. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | none |
| 3 | T3 | none |
| 4 | T4 | T1, T2, T3 |
| 5 | T5 | T1, T2, T3, T4 |
| 6 | T6 | T5 |

## Waves

| Wave | Tasks | Why together |
|------|-------|--------------|
| 1 | T1, T2, T3 | independent: the runner, the builds, the tasks |
| 2 | T4 | reads what wave 1 writes |
| 3 | T5, T6 | paid runs, in order, Zy's |

| Task | Goal | Status |
|------|------|--------|
| T1 | a run pins its mrw, its instructions and its spend | pending |
| T2 | M and N are two builds of one tag that differ in one refusal | pending |
| T3 | 48 tasks in four classes | pending |
| T4 | the counting script | pending |
| T5 | the canaries and the uncounted smoke | pending |
| T6 | the counted pilot and its report | pending |

Status: `pending` | `partial` | `blocked` | `done`.
