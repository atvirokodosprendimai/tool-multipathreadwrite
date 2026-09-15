# ADR-023 — Tasks

**Source of truth:** the task files' headers. This README is a derived index.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | none |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | The read result drops its structured envelope | done | — | Enforced-by + §61 |
| T2 | file the Desktop envelope probe recipe | done | F-6, F-16, UC3-S1, UC3-S2 | two named PASS lines; Verification Log recipe |

## Inter-task Contracts

| Contract | Produced by | Consumed by | Breaking? |
|---|---|---|---|
| `mrw_read` results with no `structuredContent` and no declared `outputSchema`; the receipt at `content[1]` | T1 | — | Yes for a caller reading `result.structuredContent` off `mrw_read`; `content[1]` is the same object |
| Desktop envelope probe recipe | T2 | — | No |

Status: `pending` | `partial` | `blocked` | `done`.
