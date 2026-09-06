# ADR-026 Tasks

Implementation tasks for ADR-026: An address may say how many lines follow. See the parent ADR for
the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins and the
README must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T3 | T1, T2 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | A read address takes a relative end | pending | — | `go test ./internal/read/ -run 'TestARelativeEnd' …` |
| T2 | A plan address takes the same relative end | pending | — | `go test ./internal/plan/ -run 'TestAPlanAddressTakesARelativeEnd' …` |
| T3 | The contract drives both paths, and the docs say the form exists | pending | — | `grep -q '^# 64\. ' scripts/contract.sh && ./scripts/contract.sh` |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | `read.Range.RelEnd` and the `A,+N` read grammar, with its two refusals | T2, T3 | T1 before T2 — T2 reuses the refusal wording so the two paths refuse the same thing in the same words, which is what makes one grammar rather than two |
| T2 | `plan.ParseAddr` accepting `A,+N` | T3 | T2 before T3 — §64 drives the built binary through both paths, so both must exist before the row can be red for the right reason |

## Notes

- The engine go/no-go is **lifted for `internal/read` and `internal/plan`**, which ADR-026 `Governs:`, and for `internal/apply`, whose test T2 extends. `internal/seen`, `internal/check` and `internal/state` stay byte-identical against the merge base, and `go.mod` declares exactly one requirement.
- The read path and the plan path must refuse the same two shapes (`+N` with no start, and `,+0`) in the same words. A divergence there is the defect this record exists to remove, and only §64 can see it — each package's own tests pass while the two disagree.
- `internal/adversarial/record_test.go` turns `go test ./...` red while a Tests table names a test that does not exist, so the repository-wide run is red from the moment these task files land until T1 and T2's tests do. That is red-first working, not a broken tree.
