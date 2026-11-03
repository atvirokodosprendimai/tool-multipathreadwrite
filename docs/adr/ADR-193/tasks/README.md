# ADR-193 Tasks

Implementation tasks for ADR-193: Claude hosts are taught to edit in search_replace blocks, if a paired pilot says it pays. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | none |
| 3 | T3 | T1, T2 |
| 4 | T4 | T3 |
| 5 | T5 | T4 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | configuration S and the stage8 profile | pending | — | `go test ./internal/bench/ ./cmd/bench/ -run TestConfigurationSIsBPlusTheEditParagraph…` |
| T2 | the frozen counting script and Stage 5's reference row | pending | — | `python3 -B -m unittest … test_bench_edits.py`; the reference row names the script's sha |
| T3 | the canary and an uncounted smoke freeze the paragraph | pending | — | the recorded paragraph hashes to its sha and is in the code; both canaries passed with B and S |
| T4 | the counted pilot and its verdict | pending | — | the verdict reproduces from the committed counts and receipts under the unchanged script |
| T5 | on SHIP, mrw instructions --host claude prints the measured paragraph | pending | — | `go test ./cmd/mrw/ -run TestInstructionsForClaudeIsTheTextConfigurationSMeasured…`; section 333 |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | configuration S, profile `stage8`, `EditParagraph` | T3, T4, T5 | T1 first |
| T2 | `bench_edits.py` and its rule | T3, T4 | T2 before T3 |
| T3 | the frozen paragraph and its sha | T4, T5 | T3 before T4 |
| T4 | the verdict line | T5 | T5 only on SHIP |

## Notes

- T4 is the paid run (about $9 and an hour of attempt time); load the costly-runs skill first. Its order against ADR-192's Stage 7 is Zy's call (T4's Awaiting-decision).
- T5 is blocked unless T4 says SHIP; on NO SHIP it is closed blocked and the guide is unchanged.
