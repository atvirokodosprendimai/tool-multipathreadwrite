# Task ADR-020-T2: A target the instruction does not name

**Depends-on:** T1
**Covers:** none — no spec
**Estimated scope:** S (one selector in the generator, one test, one contract row)
**Owner:** unassigned
**Produces:** a second target selector, and a trial id that tells the two apart
**Consumes:** `Generate`, `render`, `instruction`, `trialID` (T1, otherwise unchanged)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the instruction names no service`, `the odd block is the target`, `two selectors are two trials`, `the named fixture is unchanged`, `a legacy trial id still regenerates`, `the built binary generates one`

## Goal

The generator can produce a cell whose target is identified by a RELATION between blocks rather than
by a unique name, so a client cannot reach it by looking up one string and must compare values across
blocks. The existing named selector is unchanged and stays the default.

## Why this task exists, in the record's own words

`serviceNames` already carries the finding this task acts on: *"The threat to validity the
pre-registration names first is a target that is a unique string, because that measures matching, not
reading."* The first reading then measured exactly that — 42 correct addresses in 45 trials, flat from
2,000 to 200,000 served bytes, with every client reading the whole window. **A curve cannot bend
against a task nobody fails.** The named selector measures matching and is kept for that reason; this
one measures reading.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/curve/cell.go` | edit | the `Selector` type, the odd-retries render, the instruction that names no service, and the trial id that includes the selector |
| `internal/curve/cell_test.go` | new | **the task's Enforced-by** — the instruction names no service, the odd block is the target, and two selectors are two trial ids |
| `cmd/curve/main.go` | edit | `-selector` on `generate` |
| `scripts/contract.sh` | edit | §58 — the built binary generates a relational cell whose instruction names nothing |

## Ordered Steps

1. [S1] Write the failing test first (TDD red): with `Selector: ByOddRetries`, the instruction
   contains no service name from the fixture; exactly one block's `retries` differs from every other;
   the answer's line is that block's timeout line; and a cell that differs only in selector gets a
   different `trial_id`. [proof: acceptance]
2. [S2] Add the `Selector` type with `ByName` (the zero value, so every existing caller keeps its
   behaviour) and `ByOddRetries`. Reject any other value in `check`. [proof: mutation]
3. [S3] In `ByOddRetries`, render the target block with a different `retries` value and every other
   block with the common one, and write an instruction that describes the property and names no
   service. [proof: mutation]
4. [S4] Include the selector in `trialID`, so a result answering a named cell cannot be scored against
   the relational cell of the same size, position and seed. [proof: mutation]
5. [S5] Add contract §58: the built `curve` binary generates a relational cell, and the instruction it
   writes contains none of the service names in the fixture — paired with a named cell of the same
   parameters, whose instruction DOES name one. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
test -z "$(gofmt -l .)" \
  && go vet ./... \
  && go test ./internal/curve/ -v 2>&1 | tee /tmp/adr020-t2.out \
  && ! grep -qE 'no tests to run|no test files|^FAIL|^--- FAIL' /tmp/adr020-t2.out \
  && [ "$(grep -cE '^--- PASS: (TestATargetChosenByRelationNotName)\b' /tmp/adr020-t2.out)" = "1" ] \
  && grep -q '^# 58\.' scripts/contract.sh \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/mcp)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/mcp \
  && go test ./... \
  && ./scripts/contract.sh
```

Every clause was run BEFORE this fence was written and returned **zero hits**: the test name,
`# 58.`, and the Go identifiers `Selector` and `ByOddRetries`. **§58**: 57 is the highest on `main`.

**Every engine directory is in the go/no-go clauses**, `internal/mcp` included: this record governs
`internal/curve` and `cmd/curve` only, and a change that reaches the engine has left its remit.

⚠ **THE UNIT TEST ALONE CANNOT PROVE THIS.** A selector can be correct in `Generate` and unreachable
from the command line — which is exactly how a fixture mode nobody can generate would ship green.
§58 drives the BUILT binary and reads the instruction it wrote.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestATargetChosenByRelationNotName` | `internal/curve/cell_test.go` | **the task's Enforced-by** — the instruction names no service, exactly one block's retries differs, the answer names that block's timeout line, and two selectors are two trial ids | — | S1, S2, S3, S4 |
| `TestTheNamedSelectorIsUnchangedByTheRelationalOne` | `internal/curve/cell_test.go` | the zero `Selector` is `ByName`, the named fixture keeps its block count and its instruction still names a service | — | S2 |
| `TestALegacyTrialIDStillRegenerates` | `internal/curve/cell_test.go` | the trial id recorded in `docs/curve/reading-02-scores` regenerates from the same parameters, so a reading taken before T2 is not invalidated | — | S4 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test above |
| 2 — something selects it | `-selector` on `curve generate`; §58 drives it through the built binary |
| 3 — the caller can discover it | `curve generate -h` lists the flag and its two values |
| 4 — it is used | nothing counts generator invocations and nothing will (ADR-009). The evidence for building it is the first reading: 42/45 at a ceiling, published in `docs/curve/reading-02-result.md`, with the ceiling named there as what stops the curve bending |

## Mutation Log
<!-- filled during execution -->
- 2026-09-04 · faf26a4* · mutant killed · exit 1 · `internal/curve/cell.go` · S2: make the zero Selector something other than ByName. TestTheNamedSelectorIsUnchangedByTheRelationalOne fails — every caller written before T2 would silently get a different fixture, which is what makes the two readings incomparable. · acceptance-sha256:e76a3785553bd56e41c6ceb9d1c3bcb999ed7ebeb27cb76ff0670b5ce4f3aff8 · covers:the instruction names no service
- 2026-09-04 · faf26a4* · mutant killed · exit 1 · `internal/curve/cell.go` · S2, re-run against the mechanism it actually proves: make the zero Selector something other than ByName. TestTheNamedSelectorIsUnchangedByTheRelationalOne fails. The earlier row for this same mutant carries covers:the instruction names no service, which was the closest mechanism declared at the time; the accurate one was added to Rests-on rather than hand-editing a tool-written row. · acceptance-sha256:e76a3785553bd56e41c6ceb9d1c3bcb999ed7ebeb27cb76ff0670b5ce4f3aff8 · covers:the named fixture is unchanged
- 2026-09-04 · faf26a4* · mutant killed · exit 1 · `internal/curve/cell.go` · S3: render every block with the common retry budget, so no block is odd. TestATargetChosenByRelationNotName fails at "want one odd retries value and one common one" — the instruction would describe a property the fixture does not have, and every trial would be unanswerable rather than hard. · acceptance-sha256:e76a3785553bd56e41c6ceb9d1c3bcb999ed7ebeb27cb76ff0670b5ce4f3aff8 · covers:the odd block is the target
- 2026-09-04 · faf26a4* · mutant killed · exit 1 · `internal/curve/cell.go` · S3: fall back to the named instruction under every selector. TestATargetChosenByRelationNotName fails on the first clause — the fixture would be relational and the instruction would still hand over the name, which is the ceiling this task exists to remove and would look green. · acceptance-sha256:e76a3785553bd56e41c6ceb9d1c3bcb999ed7ebeb27cb76ff0670b5ce4f3aff8 · covers:the instruction names no service
- 2026-09-04 · faf26a4* · mutant killed · exit 1 · `internal/curve/cell.go` · S4: drop the selector from the trial id. TestATargetChosenByRelationNotName fails on clause 4 and contract section 58 fails with it — a named cell and a relational cell with the same size, position and seed would share an id, so the scorer would accept a result answering the easy trial as an answer to the hard one, and the comparison this whole task exists for would be silently wrong. · acceptance-sha256:e76a3785553bd56e41c6ceb9d1c3bcb999ed7ebeb27cb76ff0670b5ce4f3aff8 · covers:two selectors are two trials
- 2026-09-04 · faf26a4* · mutant killed · exit 1 · `cmd/curve/main.go` · S5: accept -selector odd-retries on the command line and generate the NAMED cell anyway. Every Go test still passes, because Generate is correct; only contract section 58 fails, which is the whole reason the row drives the built binary. This is the shape the fence warns about: a selector correct in the library and unreachable from the CLI would ship green. · acceptance-sha256:e76a3785553bd56e41c6ceb9d1c3bcb999ed7ebeb27cb76ff0670b5ce4f3aff8 · covers:the built binary generates one
- 2026-09-05 · f505fd0* · mutant killed · exit 1 · `internal/curve/cell.go` · S4, the defect a review found in the first version of this task: append the selector to the trial id UNCONDITIONALLY. TestALegacyTrialIDStillRegenerates fails — the same parameters that produced 96bbcee067ba in reading 2 now produce 173df413e7a5, so every raw result from a reading taken before T2 would be refused by the scorer. Nothing else fails: the fixtures are byte-identical and every other test passes, which is why it shipped in the first version and was caught by reading the committed scores rather than the code. · acceptance-sha256:e76a3785553bd56e41c6ceb9d1c3bcb999ed7ebeb27cb76ff0670b5ce4f3aff8 · covers:a legacy trial id still regenerates
- 2026-09-05 · f505fd0* · mutant killed · exit 1 · `internal/curve/cell.go` · S4, in the shipped form of trialID: drop the selector from the id entirely. TestATargetChosenByRelationNotName fails on clause 4 and contract section 58 fails with it — a named and a relational cell of the same size, position and seed would share an id, so the scorer would accept an answer to the easy trial as an answer to the hard one. This replaces the earlier S4 row, whose --from text no longer exists in the file after the legacy-id fix. · acceptance-sha256:e76a3785553bd56e41c6ceb9d1c3bcb999ed7ebeb27cb76ff0670b5ce4f3aff8 · covers:two selectors are two trials

## Invariants

- `ByName` is the zero value, so every existing caller keeps its fixture AND its trial id: the same
  parameters regenerate the id recorded in `docs/curve/reading-02-scores`, pinned by
  `TestALegacyTrialIDStillRegenerates`. The selector joins the id only when it is not `ByName`.
- In `ByOddRetries` the instruction contains no service name from the fixture.
- Exactly one block's `retries` differs; the answer's line is that block's timeout line.
- Two cells differing only in selector have different trial ids.
- Every engine directory is byte-identical; `go.mod` declares exactly one requirement.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The relational task is still findable by one grep | Med | Med | `retries` appears in EVERY block, so a search returns all of them and the client must compare values rather than jump to a match. That is weaker than "unreadable by search" and the record says so rather than claiming more |
| Two blocks make "the odd one" ambiguous | Low | High | `check` already requires ≥2 distractors, so ≥3 blocks; with 2 blocks each differs from the other and the property names nothing. The existing guard covers it and the test asserts ≥3 blocks in this mode |
| A relational result scored against a named cell | Low | High | S4 puts the selector in the trial id, and the scorer already refuses a result whose trial id does not match |

## Stop Condition

Stop if making the target unfindable requires changing what `mrw` reads, serves or applies. This
record's Served-path change is `none`, and a fixture that needs the engine to move has stopped being
a fixture.

## Out of Scope

- Running the reading with this fixture (deferred: docs/adr/BACKLOG.md — the pre-registration governs it, and a reading is a budget decision, as ADR-020's Implementation already says)
- A served window that does not begin at line 1, which would discriminate the `target+2` misses (deferred: docs/adr/BACKLOG.md — the offset-window entry)
- Any change to scoring (permanent: boundary: T1 owns the scorer, and the primary DV is unchanged)
- Making the fixture resistant to search in general (permanent: fact: `retries` appears in every block so a search returns them all; citation: this task's Risks)

## Verification Log
<!-- filled during execution -->
- 2026-09-04 · faf26a4* · exit 0 · `set -o pipefail …` · acceptance-sha256:e76a3785553bd56e41c6ceb9d1c3bcb999ed7ebeb27cb76ff0670b5ce4f3aff8 · ms:26926
- 2026-09-04 · faf26a4* · exit 0 · `set -o pipefail …` · acceptance-sha256:e76a3785553bd56e41c6ceb9d1c3bcb999ed7ebeb27cb76ff0670b5ce4f3aff8 · ms:34568
- 2026-09-04 · faf26a4* · exit 0 · `set -o pipefail …` · acceptance-sha256:e76a3785553bd56e41c6ceb9d1c3bcb999ed7ebeb27cb76ff0670b5ce4f3aff8 · ms:25314
- 2026-09-04 · faf26a4* · exit 0 · `set -o pipefail …` · acceptance-sha256:e76a3785553bd56e41c6ceb9d1c3bcb999ed7ebeb27cb76ff0670b5ce4f3aff8 · ms:25047
- 2026-09-04 · faf26a4* · exit 0 · `set -o pipefail …` · acceptance-sha256:e76a3785553bd56e41c6ceb9d1c3bcb999ed7ebeb27cb76ff0670b5ce4f3aff8 · ms:25206
- 2026-09-04 · faf26a4* · exit 0 · `set -o pipefail …` · acceptance-sha256:e76a3785553bd56e41c6ceb9d1c3bcb999ed7ebeb27cb76ff0670b5ce4f3aff8 · ms:25485
- 2026-09-05 · f505fd0* · exit 0 · `set -o pipefail …` · acceptance-sha256:e76a3785553bd56e41c6ceb9d1c3bcb999ed7ebeb27cb76ff0670b5ce4f3aff8 · ms:27317
- 2026-09-05 · f505fd0* · exit 0 · `set -o pipefail …` · acceptance-sha256:e76a3785553bd56e41c6ceb9d1c3bcb999ed7ebeb27cb76ff0670b5ce4f3aff8 · ms:25948
