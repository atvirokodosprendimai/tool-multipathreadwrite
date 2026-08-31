# ADR-001 Tasks

Implementation tasks for ADR-001: A plan addresses the original file, and
applies whole or not at all. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` /
`Covers` headers. This README is a derived index — when it disagrees with a task
file, the task file wins and the README must be regenerated.

**This ADR is a retrofit.** All three tasks describe behaviour that already
shipped (v0.0.1, v0.0.2). They are `pending` because *this corpus* holds no
tool-written evidence for them, not because the code is missing. Executing them
means running each Acceptance fence and recording a killed mutant — which
retro-proves the shipped implementation rather than rebuilding it.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T3 | T1, T2 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | The plan document parses into hunks, and reports every syntax error at once | pending | — | `go test ./internal/plan/ -run 'TestParse\|TestQuoted\|TestExplicitBody' …` |
| T2 | Every address resolves against the original file, so earlier hunks never shift later ones | pending | — | `go test ./internal/apply/ -run 'TestEarlierHunksNeverShiftLaterAddresses\|…' …` |
| T3 | One failed hunk aborts the run, and every hunk and every addressed file is reported | pending | — | `go test ./internal/apply/ -run 'TestOneBadHunkAborts…' … && ./scripts/contract.sh` |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | `plan.Parse`, `plan.Hunk`, `plan.ParseAddr` | T2, T3 | T1 before T2 and T3 |
| T2 | `apply.Apply`, `apply.Input`, `apply.Result` | T3 | T2 before T3 |

## Notes

- The TDD-red step of each task is historical and cannot be re-performed. The
  substitute this corpus accepts is a killed mutant recorded by
  `adr-verify --mutant`, which proves the test binds to the mechanism — a
  stronger claim than a red run at authoring time.
- T3's fence ends with `./scripts/contract.sh`, which builds the binary if
  `bin/mrw` is absent. It is slower than the unit fences and is deliberate: the
  abort behaviour is a property of the shipped command, not only of the package.
