# Task ADR-009-T3: Publish the first reading, and say what it does not cover

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** S (single file)
**Owner:** unassigned
**Produces:** the published authoring reading in `README.md`
**Consumes:** `mrw stats` output shape (T2)
**Data dependency:** needs a populated tally — a real corpus of `mrw write` runs from this
repository's own development. The Acceptance fence is hermetic and CANNOT see this: it checks that a
dated reading with a denominator is present, not that the reading is true. The sign-off line must
record the sample size and the date the tally was taken.
**Proof map:** v1
**Rests-on:** `the dated figure`, `the stated denominator`, `the named population`

## Goal

The README carries a real reading of how often a plan handed to mrw fails to parse, with its sample
size, its date, and an explicit statement of the population it came from.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `README.md` | edit | the reading, beside the measurement section that already exists — this is where this project's numbers are read |
| `docs/adr/ADR-009-.../tasks/T3-first-reading.md` | edit | the sign-off line recording sample size and date |

## Ordered Steps

1. [S1] Write the failing check first (TDD red): a grep-based assertion in the Acceptance fence that
   `README.md` carries a dated authoring reading with a denominator. It fails today, because the
   section does not exist. [proof: acceptance]
2. [S2] Run `mrw stats` against this repository's own tally and record the numbers verbatim — the
   counts, the denominator, the date, and the machine. A test asserting a value would pin a figure
   that goes stale the next time anyone writes a plan. [proof: human: a person runs the command against the live corpus and reads what it says]
3. [S3] Write the reading into `README.md` beside the existing measurement table, in the form the
   section already uses: the number, what produced it, and what it does not cover. [proof: acceptance]
4. [S4] State the population in the same breath: which repository, which callers, over what period.
   The parent ADR's 5% criterion is valid FOR a population, and a reading quoted without one is the
   failure mode `measure.sh` already warns about — "re-run it rather than quoting this table". [proof: acceptance]
5. [S5] Say what the reading does NOT show. It cannot distinguish a model that cannot author the
   format from a model that authored a correct plan for a file that had moved; the tally counts
   parse refusals, and only those are about the format. [proof: human: a reader must be told what a number excludes, and no test asserts the absence of a misreading]
6. [S6] Record the sign-off line in the Verification Log below: sample size, date, machine.
   [proof: human: the data dependency is non-hermetic and the fence cannot see it]

## Acceptance

```bash
set -o pipefail
grep -q '### The first reading' README.md \
  && grep -qi 'pre-registered criterion' README.md \
  && grep -qi 'under-counts by construction' README.md \
  && grep -qE 'of [0-9]+ plans' README.md \
  && ./scripts/contract.sh \
  && go test ./...
```

⚠ EVERY CLAUSE NAMES SOMETHING ONLY THIS TASK ADDS, and it did not at first. The
fence originally read `grep -qE 'measured 2026-[0-9]{2}-[0-9]{2}' README.md` —
satisfied by a line the MSYS work wrote earlier, so it would have passed on an
untouched README. That is the THIRD fence found in one session asserting
pre-existing content, after ADR-007-T3's `^# 15\.` and T1's filter that skipped
two of its own named tests. The mutation below binds the fence to this task's
own section: rename the heading and it goes red.

`of <N> plans` is the denominator clause: it fails on a bare percentage, which is the one form of
this reading that must never ship. The fence cannot check the number is TRUE — that is what the
non-hermetic data dependency and the sign-off line are for, and saying so here is the point.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| — | — | no unit test: the deliverable is a documented measurement, proved by the fence's grep and by the recorded sign-off | — | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the README section, asserted by the Acceptance fence |
| 2 — something selects it | n/a: prose has no call site. The fence's two greps are what fail when it is removed |
| 3 — the caller can discover it | it IS the discovery surface — the README is where this project's numbers are read |
| 4 — it is used | **this task closes rung 4 for ADR-001..008.** The reading is the first evidence that the plan format is authorable, which is the thing 13 tasks left as "nothing measures this yet" |

## Mutation Log

- 2026-09-03 · 2b71e2a* · mutant killed · exit 1 · `README.md` · T3 adds this section; without it the fence must go red — proving the fence binds to what this task wrote rather than to pre-existing README text · acceptance-sha256:cc5afe5fffc3779e0a1e9071003d8c0356d0e1f82cae58542e6dbff15860f086

## Invariants

- No rate is published without its denominator, its date, and its population.
- The reading says what it excludes.
- `measure.sh`'s existing numbers are untouched; this adds a reading, it does not restate theirs.

## Risks

- The first reading is from one repository and one family of callers, and will be quoted as though
  it were general. Mitigated by S4, and by the parent ADR's criterion being scoped to a population.
- A flattering first number ends curiosity. Mitigated by S5 naming what the tally cannot see, and by
  the Follow-up in the parent ADR asking for a second reading.

## Stop Condition

Stop if the tally's sample is too small to report — under 30 plans, a percentage is noise wearing a
decimal point. Say so in the README instead of publishing a rate, and leave this task `partial` with
the sample size recorded. A number nobody should act on is worse than an admission.

## Out of Scope

- A cross-model comparison (deferred: docs/adr/BACKLOG.md)
- Changing the plan format in response to the reading (deferred: docs/adr/BACKLOG.md)

## Verification Log
- 2026-09-03 · 2b71e2a* · exit 0 · `set -o pipefail …` · acceptance-sha256:05ca3a48793b523a5d9bee3743f77ea6a2dbdeffc0bdf6bbb222b60fb5f5b27d · ms:10670
- 2026-09-03 · human-observed · sample taken 2026-09-03 on this repository, Apple M5 (darwin/arm64): 9 plans, 9 applied, 0 refused. BELOW the 30-plan floor this task sets, so the README publishes a count and an admission rather than a rate, and this task stays partial. Population is one repository, one model, one session — the narrowest possible.
- 2026-09-03 · 2b71e2a* · exit 0 · `set -o pipefail …` · acceptance-sha256:cc5afe5fffc3779e0a1e9071003d8c0356d0e1f82cae58542e6dbff15860f086 · ms:9599
- 2026-09-03 · 2b71e2a* · exit 0 · `set -o pipefail …` · acceptance-sha256:cc5afe5fffc3779e0a1e9071003d8c0356d0e1f82cae58542e6dbff15860f086 · ms:8493
- 2026-09-04 · b80bf7e · exit 0 · `set -o pipefail …` · acceptance-sha256:2f495d1b3e4606639002f045113c8a87d92aeb90e35688a562ae6fb292655dbd · ms:11330
