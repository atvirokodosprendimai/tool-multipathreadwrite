# ADR-055 Tasks

Implementation tasks for ADR-055: The receipt counts its advisories, notices a pattern, and can refuse the wrap-tail shape. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T3 | none |
| 4 | T4 | T1, T2, T3 |

M's order by value per unit of work: advisory count, repeat pattern, strict balance. T3 may run
in parallel with T1/T2. T4 teaches once the three surfaces exist.

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | Advisory count on the summary line and the receipt; contract §92 | pending | — | `docs/adr/ADR-055-the-receipt-counts-its-advisories-and-notices-a-pattern/tasks/T1-advisory-count-and-contract-92.md` fence |
| T2 | Recent-window ring and the pattern line; contract §93 | pending | — | `docs/adr/ADR-055-the-receipt-counts-its-advisories-and-notices-a-pattern/tasks/T2-repeat-pattern-and-contract-93.md` fence |
| T3 | `--strict-balance` opt-in refusal; contract §94 | pending | — | `docs/adr/ADR-055-the-receipt-counts-its-advisories-and-notices-a-pattern/tasks/T3-strict-balance-and-contract-94.md` fence |
| T4 | Teach the three; BACKLOG pre-registration | pending | — | `docs/adr/ADR-055-the-receipt-counts-its-advisories-and-notices-a-pattern/tasks/T4-teach-and-backlog.md` fence |

Status: `pending` | `partial` | `blocked` | `done`.

Contract rows (§92–§94) are cited in each task's Ordered Steps and Acceptance fence, NOT in its
Tests table: a `§NN` row is a script section, not a function, and the first-red lock (ADR-050 in the
harness) records it `unproven` forever, refusing `done`. Learned on ADR-054; see the `first-red-lock`
skill. Un-strike the Tests rows BEFORE the first `adr-verify`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | `apply.Result.Advisories` | T2, T4 | T1 before T2 |
| T2 | `authoring.Recent` + pattern line | T4 | T2 before T4 |
| T3 | `Options.StrictBalance` / `--strict-balance` | T4 | T3 before T4 |

## Notes

- Branch from `origin/main`.
- Engine go/no-go: `internal/check`, `internal/read`, `internal/plan`, `internal/seen`, `internal/state` stay byte-identical. `internal/apply` is owned by T1 (count) and T3 (refusal); `internal/authoring` vocabulary stays five names (T2 adds a file and readers, not an Outcome).
- Do not start T1 while the parent is Proposed.
