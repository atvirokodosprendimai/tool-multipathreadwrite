# Task ADR-009-T2: `mrw stats` prints the tally, sample size beside the rate

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** unassigned
**Produces:** `mrw stats`, `mrw stats --json`, `mrw stats --reset`
**Consumes:** `authoring.Load`, `authoring.Tally` (T1)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the exit code`, `the sample size printed beside the rate`, `the empty-tally wording`

## Goal

A caller can read the tally, as a rate with its denominator, and can empty it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `cmd/mrw/main.go` | edit | the `stats` subcommand, its two flags, and **its registration in the command list** — the line that makes it reachable |
| `cmd/mrw/planpath_test.go` | edit | CLI-level tests |
| `README.md` | edit | the subcommand and what the number means; a surface a caller cannot discover is rung 3 unmet |
| `scripts/contract.sh` | edit | §37: the three invocations through the real binary, including `--reset` |

## Ordered Steps

1. [S1] Write the failing tests first (TDD red): `stats` on a fresh checkout prints zeros and says so
   in words; after a recorded write it prints a non-zero `applied`; `--reset` empties it; `--json`
   parses.
2. [S2] Add the `stats` subcommand and **register it**, so `mrw stats` resolves and `mrw help` lists
   it. Registration is the selecting line — without it the action is unreachable code with tests.
3. [S3] Print the rate WITH its denominator on the same line — `refused_parse 2 of 137 plans (1.5%)`,
   never a bare percentage. The parent ADR's criterion is valid for the population that produced it,
   and a percentage without its sample size is the form that gets quoted out of that context.
   [proof: acceptance]
4. [S4] Say plainly what a zero tally means: that nothing has been recorded yet, not that nothing has
   failed. An empty measurement that reads like a good result is the silent-success shape this
   project refuses. [proof: acceptance]
5. [S5] Add `--json` emitting the same numbers, and `--reset` emptying the tally and saying how many
   records it discarded — a reset that reports nothing is indistinguishable from a reset that did
   nothing.
6. [S6] Add contract §37 driving each of the three through the real binary. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./cmd/mrw/ -run 'TestStats' -v 2>&1 | tee /tmp/adr009-t2.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr009-t2.out \
  && go test ./... \
  && grep -q '^# 37\.' scripts/contract.sh \
  && grep -q 'mrw stats' README.md \
  && ./scripts/contract.sh
```

The `README.md` clause is how rung 3 is proved rather than asserted: a subcommand nobody documents
is one the caller cannot discover, and the fence is the only thing that notices.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestStatsOnAFreshCheckoutSaysNothingIsRecordedYet` | `cmd/mrw/planpath_test.go` | S4 — zero is not reported as success | — | S1, S4 |
| `TestStatsPrintsTheDenominatorBesideTheRate` | `cmd/mrw/planpath_test.go` | S3 — no bare percentage | — | S1, S3 |
| `TestStatsCountsARecordedWrite` | `cmd/mrw/planpath_test.go` | the read path sees what T1 wrote | — | S1, S2 |
| `TestStatsResetEmptiesAndReportsWhatItDiscarded` | `cmd/mrw/planpath_test.go` | S5 — a silent reset is indistinguishable from a no-op | — | S1, S5 |
| `TestStatsJSONParses` | `cmd/mrw/planpath_test.go` | `--json` is machine-readable | — | S5 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the five tests above |
| 2 — something selects it | the command registration in `cmd/mrw/main.go` (S2); deleting it makes `TestStatsCountsARecordedWrite` fail, and that test is inside the Acceptance fence |
| 3 — the caller can discover it | `mrw help` lists it, `README.md` documents it, and the fence asserts the README changed |
| 4 — it is used | T3 publishes the first reading; nothing counts invocations of `stats` itself, and that is deliberate — counting the counter is where this stops |

## Mutation Log

## Invariants

- `stats` reads and never writes, except under `--reset`.
- A rate is never printed without its denominator.
- Nothing in the output identifies a file, a plan or a caller.

## Risks

- `--reset` is destructive and one keystroke from `--json`. Mitigated by it reporting what it
  discarded, so an accidental reset is visible rather than silent.
- A rate on a tiny sample invites over-reading. Mitigated by S3, and by the parent ADR stating the
  criterion is valid for the population that produced it.

## Stop Condition

Stop if `stats` needs to read anything outside the tally to be useful — the working tree, the ledger,
git. The tally is either a sufficient record of what happened or it is the wrong record, and joining
it to other state would make a reporting surface into a second source of truth about writes.

## Out of Scope

- Publishing a reading — T3's job.
- Counting invocations of `stats` itself (permanent: boundary: counting the counter has no reader)

## Verification Log
