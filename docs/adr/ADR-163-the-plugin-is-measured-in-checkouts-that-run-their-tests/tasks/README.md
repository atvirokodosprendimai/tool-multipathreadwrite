# ADR-163 Tasks

Implementation tasks for ADR-163: bench runs SWE-bench attempts in checkouts that can run their tests, for quality-harness ADR-112. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T3 | T2 |
| 4 | T6 | T2 |
| 5 | T4 | T3, T6 |
| 6 | T5 | T4 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | the instance's container and the shims | pending | — | `go test ./internal/bench/ -run '^(TestStage6ShimsRunInTheInstanceContainer…)$'` |
| T2 | configuration W, the declared check and the stage's caps | pending | — | `go test ./internal/bench/ -run '^(TestConfigurationWIsPPlusTheWorkLine…)$'` |
| T3 | the baseline, the surviving pool and the block plan | pending | — | `go test ./internal/bench/ -run '^(TestTheBaselineRecordsGreenRedAndTime…)$'` |
| T4 | the baseline pilot, the whole-pool baseline, then the agent pilot | pending | — | `jq … docs/bench/stage-6-pilot/baseline.json …` |
| T5 | the stage, block by block, and its results | pending | — | `go run ./cmd/bench verify --stage stage6 --tasks "$t" …` |
| T6 | the side arms G and Q, read against P | pending | — | `go test ./internal/bench/ -run '^(TestConfigurationGIsPWithTheGoHooksBinary…)$'` |

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | the container, the shims, `--profile stage6`, `host_interpreter_calls` | T2, T3, T4, T5 | T1 before T2 |
| T2 | W, `checks.json`, the declared check, `--max-turns` | T3, T4, T5 | T2 before T3 |
| T3 | `bench baseline`, the surviving pool file, its blocks, `--pair A,W` against it | T4, T5 | T3 before T4 |
| T6 | configurations G and Q, the staged binary, the switch, `--pair P,G`/`--pair P,Q` | T4, T5 | T6 before T4 |
| T4 | D1's answer, the surviving pool, D3's caps, the price | T5 | T4 before T5 |

## Notes

- Zy, 2026-10-19: "Draft now, run after". T1 to T3 are hermetic code and may be built while Stage 5 runs; anything that pulls or runs an image (T1 S1, T4, T5) starts only after Stage 5 closes (ADR-162 T5's `bench look` exits 0, or Zy ends it), on Zy's word.
- The registration ("Pre-registered for ADR-163" in `docs/adr/BACKLOG.md`) is committed with this record, before any code or run.
