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
| 5 | T5 | T1, T2, T3, T4 (and ADR-211 T1, T2, T3, named in T5's Consumes) |
| 6 | T6 | T5 |

## Waves

| Wave | Tasks | Why together |
|------|-------|--------------|
| 1 | T1, T2, T3 | independent: the runner, the builds, the tasks |
| 2 | T4 | reads what wave 1 writes |
| 3 | T5, T6 | paid runs, in order, Zy's, each step gated before the next and not before Zy says so again (ADR-211 Decisions 9 and 12) |

| Task | Goal | Status |
|------|------|--------|
| T1 | a run pins its mrw, its instructions and its spend | done (#693) |
| T2 | M and N are two builds of one tag that differ in one refusal | done (#691) |
| T3 | 48 tasks in four classes | done (#692) |
| T4 | the counting script | done (#695) |
| T5 | the canaries and the uncounted smoke (amended by ADR-211 T3) | pending |
| T6 | the counted pilot and its report (amended by ADR-211 T3) | pending |

Status: `pending` | `partial` | `blocked` | `done`.
