# ADR-026 Tasks

Implementation tasks for ADR-026: An address may say how many lines follow. See the parent ADR for
the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

## Waves

Every task depends on the one before it, so the DAG is a chain and each wave holds one task. That is
not an accident of scheduling: T1-T3 built the form, and T4 through T7 are four rounds of review
findings against it, each of which could only be written once the previous round had landed.

| Wave | Tasks | Depends-on |
|------|-------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T3 | T1, T2 |
| 4 | T4 | T1, T2, T3 |
| 5 | T5 | T4 |
| 6 | T6 | T5 |
| 7 | T7 | T6 |

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T3 | T1, T2 |
| 4 | T4 | T1, T2, T3 |
| 5 | T5 | T4 |
| 6 | T6 | T5 |
| 7 | T7 | T6 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | A read address takes a relative end | done | — | `go test ./internal/read/ -run 'TestARelativeEnd' …` |
| T2 | A plan address takes the same relative end | done | — | `go test ./internal/plan/ -run 'TestAPlanAddressTakesARelativeEnd' …` |
| T3 | The contract drives both paths, and the docs say the form exists | done | — | `grep -q '^# 64\. ' scripts/contract.sh && ./scripts/contract.sh` |
| T4 | One lexer, and an address no op can half-ignore | done | — | `go test ./internal/addr/ ./internal/plan/ ./internal/apply/ …` |
| T5 | The lexer scans, and the wire teaches the form | done | — | `go test ./internal/addr/ ./internal/apply/ …` |
| T6 | One scanner for every delimiter, and no arithmetic that wraps | done | — | `go test ./internal/read/ -run 'IntegerBoundary|BackslashIsClosed' …` |
| T7 | The header does not rewrite the pattern it carries | done | — | `go test ./internal/read/ ./internal/plan/ -run 'MalformedPattern|QuoteInsideAPattern' …` |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | `read.Range.RelEnd` and the `A,+N` read grammar, with its two refusals | T2, T3 | T1 before T2 — T2 reuses the refusal wording so the two paths refuse the same thing in the same words, which is what makes one grammar rather than two |
| T2 | `plan.ParseAddr` accepting `A,+N` | T3 | T2 before T3 — §64 drives the built binary through both paths, so both must exist before the row can be red for the right reason |
| T1, T2 | the duplicated suffix lexer and its refusal wording | T4 | T4 replaces both copies with `internal/addr.CutRelative` — the Codex review of #125 measured them already divergent, which is the second occurrence `docs/adr/BACKLOG.md` pre-registered a record for |

## Notes

- The engine go/no-go is **lifted for `internal/read` and `internal/plan`**, which ADR-026 `Governs:`, and for `internal/apply`, whose test T2 extends. `internal/seen`, `internal/check` and `internal/state` stay byte-identical against the merge base, and `go.mod` declares exactly one requirement.
- The read path and the plan path must refuse the same two shapes (`+N` with no start, and `,+0`) in the same words. A divergence there is the defect this record exists to remove, and only §64 can see it — each package's own tests pass while the two disagree.
- `internal/adversarial/record_test.go` turns `go test ./...` red while a Tests table names a test that does not exist, so the repository-wide run is red from the moment these task files land until T1 and T2's tests do. That is red-first working, not a broken tree.
