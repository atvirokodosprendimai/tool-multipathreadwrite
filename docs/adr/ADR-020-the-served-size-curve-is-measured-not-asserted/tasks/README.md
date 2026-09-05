# ADR-020 — Tasks

## Task Index

| Task | Title | Status |
|---|---|---|
| T1 | The harness that generates a cell and scores a result | done |
| T2 | A target the instruction does not name | done |
| T3 | The odd retry budget is drawn, not fixed | done |
| T4 | A served window that does not begin at line one | done |

## Inter-task Contracts

| Contract | Produced by | Consumed by | Breaking? |
|---|---|---|---|
| the generator, the manifest shape, and the scorer | T1 | the first reading (a Follow-up, not a task) | No |
| a second target selector, and a trial id that tells the two apart | T2 | a later reading (a Follow-up, not a task) | No |
| a relational fixture with no constant signature | T3 | a later reading (a Follow-up, not a task) | No |
| a cell served from a chosen line, so row and line number diverge | T4 | the discriminating reading (a Follow-up, not a task) | No |
