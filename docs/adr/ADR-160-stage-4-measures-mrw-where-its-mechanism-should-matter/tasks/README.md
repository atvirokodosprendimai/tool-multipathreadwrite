# ADR-160 Tasks

Implementation tasks for ADR-160: Stage 4 measures mrw where its mechanism should matter, on tasks chosen by rule. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T6 | T2 |
| 4 | T3 | T2, T6 |
| 5 | T4 | T3 |
| 6 | T5 | T4 (and ADR-156 task 6, which creates RESULTS.md) |

## Waves

Every task depends on the one before it, so each wave holds one task: T1, T2, T6, T3, T4, T5.

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | the selection rules in code | done | — | `go test ./internal/bench/ -run '^(TestAWideRename…)$'` |
| T2 | the Stage 4 corpus, drawn by the registered rules | done | — | `test … = 10 && … = 8 && … stage-4-tasks.txt …` |
| T6 | the pilot's fixes: B's interpreter rule, check-coupled graded by its tests, the mine floor as a flag | done | — | `go test ./internal/bench/ ./cmd/bench/ -run '^(TestAnInterpreter…)$'` |
| T3 | difficulty calibrated on A outside the sample, and harness pilots that choose nothing | pending | — | `go run ./cmd/bench verify --stage pilot …` |
| T4 | Stage 4 on Sonnet 5.5 and Opus 5.5 | pending | — | `go run ./cmd/bench verify --stage stage4 …` |
| T5 | Stage 4's verdicts in RESULTS.md, and one README sentence each | pending | — | `go run ./cmd/bench analyze --selection … docs/bench …` |

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | `bench mine`, `swebench-select --salt --min-hunks`, the shapes | T2 | T1 before T2 |
| T2 | `bench/tasks/<stage-4 ids>/`, `selection-adr-160.json` | T3, T4 | T2 before T3 |
| T6 | `test_package`, the hook's interpreter rule, `bench mine --min-occurrences --min-files` | T3, T4 | T6 before T3 |
| T3 | the calibrated corpus | T4 | T3 before T4 |
| T4 | `docs/bench/stage-4-*/` | T5 | T4 before T5 |

## Notes

- Every run starts only on Zy's word; `costly-runs` applies, and to the slice's images above all.
- The registration (T1 S2) is committed before any candidate is drawn.
