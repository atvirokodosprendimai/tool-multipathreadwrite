# ADR-033 Tasks

Implementation tasks for ADR-033: A cap of zero is a cap. See the parent ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers.
This README is a derived index — when it disagrees with a task file, the task file wins.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | A cap of zero is honoured | done | — | `go test ./internal/read/ -run 'TestACapOfZeroServesNothing' …` |
| T2 | The contract drives both spellings | done | — | `grep -q '^# 69\. ' scripts/contract.sh && ./scripts/contract.sh` |

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | `MaxLines *int` and the zero-is-a-cap rule | T2 | T1 before T2 — the row drives the built binary |

## Notes

- ⚠ **A pointer, not a `-1` sentinel.** Three call sites omit the field entirely; with a numeric sentinel their zero value would mean "serve nothing". Nil must keep meaning "no cap".
- ⚠ Both spellings in one contract section: a row asserting only the zero case is green against a binary that serves nothing at all.
- This record owns `internal/read` and `cmd/mrw`. `internal/apply`, `internal/plan`, `internal/seen` and `internal/state` stay byte-identical against the merge base.
