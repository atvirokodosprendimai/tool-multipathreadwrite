# ADR-035 Tasks

A multi-line `replace` must carry `anchor=`. Two tasks: the guard and its Go
tests, then the contract row that drives the built binary and the documentation
that stops promising the old grammar.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | A multi-line replace is refused without an anchor | done | — | `go test ./internal/apply/ -run 'TestAMultiLineReplaceWithoutAnAnchorIsRefused…'` |
| T2 | The contract drives the refusal, and the docs teach it | done | — | `grep -q '^# 73\. ' scripts/contract.sh && ./scripts/contract.sh` |

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | the refusal in `apply.Apply`, and its message naming `anchor=` | T2 | T1 before T2 — §73 drives the built binary and has nothing to assert otherwise |
