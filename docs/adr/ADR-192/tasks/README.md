# ADR-192 Tasks

Implementation tasks for ADR-192: The read hook is measured on SWE-bench, as a configuration H read against B and A. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T3 | T1, T2 |
| 4 | T4 | T3 |
| 5 | T5 | T4 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | configuration H and the stage7 profile | pending | — | `go test ./internal/bench/ -run TestConfigurationHIsBPlusReadAndTheReadHook…` |
| T2 | bench look and analyze read H against B and A, with the uptake gate | pending | — | `go test ./internal/bench/ ./cmd/bench/ -run TestLookReadsHAgainstBAndA…` |
| T3 | the release, the re-probe, the canary, a preflight and an uncounted pilot freeze the sentence | blocked | — | the recorded sentence hashes to its sha and is in the code; both canaries passed with H |
| T4 | Stage 7, block by block | pending | — | `bench look docs/bench/stage-7` reproduces the last look and says complete |
| T5 | the verdicts, the mechanism counts and the README rows | pending | — | the six verdict lines of the last look are quoted; the counts cover every H transcript |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | configuration H, profile `stage7`, receipt `host_reads` | T2, T3, T4, T5 | T1 first |
| T2 | comparisons `H:B`, UNTESTED | T4 | T2 before T4 |
| T3 | the frozen sentence and its sha | T4 | T3 before T4 |
| T4 | the looks and transcripts | T5 | T4 before T5 |

## Notes

- T3 is blocked until a release carries ADR-191 (`1f01dcdc`); cutting it is Zy's call.
- T4 is the paid run: about 21 hours of attempt time projected, on a shared machine, over days; load the costly-runs skill first.
