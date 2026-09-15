# ADR-022 — Tasks

Implementation tasks for ADR-022: a path-scoped rule arrives on an mrw read too.

**Source of truth:** the task files' headers. This README is a derived index.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | none |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | The hook, its wiring, and the test that drives it | done | — | Enforced-by + §55 |
| T2 | the hook returns within 2 s and still exits 0 | done | F-7, UC4-S1, UC4-S2 | two named PASS lines, `# 110.`, `signal.alarm(2)` |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

None.

## Inter-task Contracts

| Contract | Produced by | Consumed by | Breaking? |
|---|---|---|---|
| the hook, its wiring, and the Enforced-by test | T1 | — | No |
| 2 s wall-clock bound, still exit 0 | T2 | — | No |
