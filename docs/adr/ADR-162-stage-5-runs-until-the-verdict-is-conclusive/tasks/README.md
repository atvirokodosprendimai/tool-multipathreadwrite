# ADR-162 Tasks

Implementation tasks for ADR-162: Stage 5 runs SWE-bench Verified until each verdict is conclusive, with the quality-harness plugin as a third configuration. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T3 | T2 |
| 4 | T4 | T3 |
| 5 | T5 | T4 |
| 6 | T6 | T5 |

## Waves

| Wave | Tasks | Depends-on |
|------|-------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T3 | T2 |
| 4 | T4 | T3 |
| 5 | T5 | T4 |
| 6 | T6 | T5 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | configuration P and the Stage 5 profile | pending | — | `go test ./internal/bench/ -run '^(TestConfigurationP…)$'` |
| T2 | the claim line, the harness's own result, and ADR-106's paired rows | pending | — | `go test ./internal/bench/ -run '^(TestTheStatusLine…)$'` |
| T3 | the pool, the blocks and the looks | pending | — | `go test ./internal/bench/ -run '^(TestTheStage5Pool…)$'` |
| T4 | the harness pilot, and room on the disk | pending | — | `go run ./cmd/bench verify --stage pilot-5 …` |
| T5 | ADR-106 amended, then Stage 5 block by block | pending | — | `go run ./cmd/bench look docs/bench/stage-5 …` |
| T6 | Stage 5's verdicts and the rows handed to quality-harness | pending | — | `go run ./cmd/bench analyze --selection … docs/bench …` |

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | configuration P, the Stage 5 profile, `--plugin-dir`, `--node`, `--codex-stub`, the receipt's `plugin` | T2, T4, T5 | T1 before T2 |
| T2 | `--status-line`, `status`, `broke_tests`, `harness_resolved`, `export-paired` | T3, T4, T5, T6 | T2 before T3 |
| T3 | `selection-adr-162.json`, `bench/repos.json`, `--block`, `--configs-for`, `bench look`, `analyze`'s stage levels | T4, T5, T6 | T3 before T4 |
| T4 | the pilot's sign-off | T5 | T4 before T5 |
| T5 | `docs/bench/stage-5/block-K/`, `looks/look-K.txt` | T6 | T5 before T6 |

## Notes

- Every run starts only on Zy's word; `costly-runs` applies, and to the images above all.
- The registration ("Pre-registered for ADR-162" in `docs/adr/BACKLOG.md`) is committed with this record, before any code or run.
