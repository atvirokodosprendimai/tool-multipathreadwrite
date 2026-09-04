# ADR-009 Tasks

Implementation tasks for ADR-009: mrw counts what happens to the plans it is given. See the parent
ADR for the decision.

**Source of truth:** the task files' `Depends-on` / `Produces` / `Consumes` / `Covers` headers. This
README is a derived index — when it disagrees with a task file, the task file wins and the README
must be regenerated.

## Execution Order

| Order | Task | Depends-on |
|-------|------|------------|
| 1 | T1 | none |
| 2 | T2 | T1 |
| 3 | T3 | T2 |

The order is forced rather than chosen: T2 reads what T1 writes, and T3 publishes what T2 prints.

**T1 carries its own call site.** It could have shipped the package alone and left `cmd/mrw` to T2,
which is the shape this repository keeps regretting — `rooted.Descendable` was built, tested,
mutation-logged and deleted unused on 2026-09-03 because its only intended caller never reached it.
So the task that adds the package is the task that wires it, and its rung 2 mutation lives inside
its own Acceptance fence.

**T3 is the one that closes rung 4**, which reads `nothing measures this yet` or `nothing counts
them` on all 13 tasks of ADR-001..008. It is last because a reading needs something to read.

## Task Index

| ID | Title | Status | Covers | Acceptance |
|----|-------|--------|--------|------------|
| T1 | Record the outcome of every plan, in counts only | done | — | `go test ./internal/authoring/ -run 'TestTally\|TestTheTally' … && ./scripts/contract.sh` |
| T2 | `mrw stats` prints the tally, sample size beside the rate | done | — | `go test ./cmd/mrw/ -run 'TestStats' … && ./scripts/contract.sh` |
| T3 | Publish the first reading, and say what it does not cover | done | — | `grep -q '### The first reading' README.md && grep -qi 'pre-registered criterion' README.md && … && go test ./...` |

**T3 stayed `partial` until the tally crossed its own floor, and the Stop Condition is why.** The
first reading was 9 plans — below the 30-plan floor this task sets for publishing a RATE — so the
README carried a count and an admission instead of "100%", and the task stayed open.
It is `done` as of 2026-09-04 at 68 recorded outcomes: 65 applied, 2 refused_apply, 1 refused_parse.

The floor was not crossed by more work. It was crossed by rebuilding the authoring machine's binary:
it had been running v0.0.14, which predates the recorder entirely, so every plan it applied was
invisible to the tally. 9 became 68 in a day. That is now the sharpest caveat in the published
section, because it says the denominator is biased toward callers who already upgraded.

Status: `pending` | `partial` | `blocked` | `done`.

## Contract Coupling

| Producer | Contract | Consumer(s) | Ordering note |
|----------|----------|-------------|---------------|
| T1 | `authoring.Outcome`, `authoring.Record`, `authoring.Load`, `authoring.Tally` | T2 | T1 before T2 — T2 reads a file whose format T1 defines |
| T2 | `mrw stats` output shape | T3 | T2 before T3 — T3 quotes what T2 prints |

## Notes

- **T3 is not hermetic and its fence cannot see that.** It needs a populated tally from real use, while
  its Acceptance only checks that a dated reading with a denominator is present in `README.md`. The
  sign-off line must record the sample size, the date and the machine, or the gate goes green on a
  number nobody took.
- **T3 has a floor.** Under 30 plans it publishes an admission rather than a rate, and stays
  `partial`. A percentage on a handful of samples is noise wearing a decimal point.
- The parent ADR's criterion — parse refusals over 5% means the FORMAT is the problem — is
  pre-registered here so the reading cannot be interpreted after the fact to mean whatever it says.
