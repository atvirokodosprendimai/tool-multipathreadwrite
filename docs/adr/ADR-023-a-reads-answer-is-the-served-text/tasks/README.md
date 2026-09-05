# ADR-023 — Tasks

## Task Index

| Task | Title | Status |
|---|---|---|
| T1 | The read result drops its structured envelope | pending |

## Inter-task Contracts

| Contract | Produced by | Consumed by | Breaking? |
|---|---|---|---|
| `mrw_read` results with no `structuredContent` and no declared `outputSchema`; the receipt at `content[1]` | T1 | — | Yes for a caller reading `result.structuredContent` off `mrw_read`; `content[1]` is the same object |

Status: `pending` | `partial` | `blocked` | `done`.
