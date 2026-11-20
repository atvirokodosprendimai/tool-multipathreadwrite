# ADR-211 Tasks

Implementation tasks for ADR-211: Stage 9 is amended to the claim its instrument can carry, and pays only after a free rehearsal. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T3 | T1 |
| 3 | T2 | T1, T3 |

## Waves

| Wave | Tasks | Why together |
|------|-------|--------------|
| 1 | T1 | the instrument |
| 2 | T3 | the paid steps' gates and the registration, which read T1's output |
| 3 | T2 | the rehearsal runs what T1 and T3 built |

| Task | Goal | Status |
|------|------|--------|
| T1 | the instrument says what it cannot see, reconciles cost, and holds its identity | done |
| T2 | a free rehearsal of everything the paid tasks rely on | partial |
| T3 | each paid step is gated before the next, and the registration says what the pilot can show | done |

Status: `pending` | `partial` | `blocked` | `done`.
