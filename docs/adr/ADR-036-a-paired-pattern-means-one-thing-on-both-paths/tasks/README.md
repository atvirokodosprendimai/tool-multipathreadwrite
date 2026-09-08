# ADR-036 Tasks

One task: the resolver change, its tests, the contract row that drives the built
binary, and the documentation that stops describing the divergence.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | A paired pattern ends at or after its start, or is refused | done | — | `go test ./internal/read/ -run 'TestAPairedPatternEndsAtOrAfterItsStart…' && grep -q '^# 74\. ' scripts/contract.sh && ./scripts/contract.sh` |

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | the resolver's at-or-after end and its refusal of a missing end | §74 | one task; the row drives the built binary in the same change |
