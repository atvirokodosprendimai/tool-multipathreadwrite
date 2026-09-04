# Task ADR-020-T3: The odd retry budget is drawn, not fixed

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** XS (one draw in the generator, one test, one contract assertion)
**Owner:** unassigned
**Produces:** a relational fixture with no constant signature
**Consumes:** `render`, `Selector` (T2, otherwise unchanged)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the odd value varies across seeds`, `the named fixture is untouched`, `the drawn pair is stable across the fit`, `the built binary draws it`

## Goal

The relational fixture's retry budgets are drawn from the trial's seed, so no value is a signature a
client can carry from one cell to the next. The named fixture is byte-identical to what it was.

## Why this task exists

T2 removed the unique NAME and left a constant VALUE. Measured on `main` at `e96504a`, every
relational cell at every seed rendered three blocks at `retries = 3` and the target at `retries = 5`:

```
seed 1:  3 × "retries = 3"   1 × "retries = 5"
seed 99: 3 × "retries = 3"   1 × "retries = 5"
```

So a client that has seen one relational cell can search `retries = 5` in every other, at a cost
independent of served size. That is the single-match shortcut T2 exists to remove, surviving inside
T2's own fixture — and it would have put the relational reading at a ceiling in the same way the
named one was, with the trials already spent before anyone noticed. **Found by review of PR #93 and
confirmed here before acting on it.**

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/curve/cell.go` | edit | `retryPair` draws the two budgets from the seed; `render` uses them only under `ByOddRetries` |
| `internal/curve/cell_test.go` | edit | **the task's Enforced-by** — the odd value differs across seeds, and the named fixture still renders the constant |
| `scripts/contract.sh` | edit | §59 — two relational cells from the built binary, different seeds, different odd values |

## Ordered Steps

1. [S1] Write the failing test first (TDD red): across a set of seeds the odd retry budget takes more
   than one distinct value, and the common one does too; and a named cell of the same parameters
   still renders `retries = 3` in every block. [proof: acceptance]
2. [S2] Add `retryPair`, deriving both budgets from the trial's seed and guaranteeing they differ.
   It must be stable across the padding-fit loop, which calls `render` repeatedly, so it is derived
   from the seed and not from a generator that advances. [proof: mutation]
3. [S3] Use the drawn pair only under `ByOddRetries`; `ByName` keeps the constant, so every named
   cell stays byte-identical and the ids recorded in `docs/curve/reading-02-scores` still regenerate.
   [proof: mutation]
4. [S4] Extend contract §59: the built binary generates two relational cells at different seeds whose
   odd values differ, paired with two NAMED cells at different seeds that are byte-identical in their
   retries lines. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
test -z "$(gofmt -l .)" \
  && go vet ./... \
  && go test ./internal/curve/ -v 2>&1 | tee /tmp/adr020-t3.out \
  && ! grep -qE 'no tests to run|no test files|^FAIL|^--- FAIL' /tmp/adr020-t3.out \
  && [ "$(grep -cE '^--- PASS: (TestTheOddRetryBudgetIsNotAFixedSignature)\b' /tmp/adr020-t3.out)" = "1" ] \
  && [ "$(grep -cE '^--- PASS: (TestTheNamedFixtureMatchesItsGoldenBytes|TestTheDrawIsStableAcrossThePaddingFit)\b' /tmp/adr020-t3.out)" = "2" ] \
  && [ "$(grep -cE '^--- PASS: (TestALegacyTrialIDStillRegenerates)\b' /tmp/adr020-t3.out)" = "1" ] \
  && grep -q '^# 59\.' scripts/contract.sh \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/mcp \
  && go test ./... \
  && ./scripts/contract.sh
```

Every clause was run BEFORE this fence was written and returned **zero hits**: the test name, `# 59.`
and the Go identifier `retryPair`. **§59**: 58 is the highest on `main`.
The legacy-id clause is in the fence deliberately, but it is NOT what protects the named fixture, and
an earlier version of this paragraph said it was. `trialID` hashes the PARAMETERS and never the
rendered file, so drifted bytes reuse the recorded id in silence — the opposite of a guard. What
protects the bytes is `TestTheNamedFixtureMatchesItsGoldenBytes`, which pins digests taken from the
binary at `origin/main`, and §59, which checks the shape through the built binary. The id clause
stays because it is cheap and pins a different thing. **Corrected after review of PR #94 named the
rationale as backwards.**

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheOddRetryBudgetIsNotAFixedSignature` | `internal/curve/cell_test.go` | **the task's Enforced-by** — across seeds the odd budget takes more than one value, the pair always differs, and a named cell still renders the constant in every block | — | S1, S2, S3 |
| `TestTheNamedFixtureMatchesItsGoldenBytes` | `internal/curve/cell_test.go` | the named fixture, served rendering and answer hash to digests taken from `origin/main`'s binary — the guard the trial id cannot be, since it hashes parameters | — | S3 |
| `TestTheDrawIsStableAcrossThePaddingFit` | `internal/curve/cell_test.go` | one cell rendered at five padding widths keeps the same budgets, so the cell measured is the cell generated | — | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test above |
| 2 — something selects it | `render` under `ByOddRetries`; §59 drives the built binary |
| 3 — the caller can discover it | nothing to discover: it is a property of the generated fixture, and the record is where it is stated |
| 4 — it is used | no relational reading has been taken yet, which is why this lands BEFORE one rather than after. The evidence for building it is the measurement in "Why this task exists" |

## Mutation Log
<!-- filled during execution -->
- 2026-09-05 · e96504a* · mutant killed · exit 1 · `internal/curve/cell.go` · S2: fix the common budget instead of drawing it. TestTheOddRetryBudgetIsNotAFixedSignature fails on the common-value clause — a constant common value identifies the odd block by elimination, which is the same shortcut one level removed, and it is the half of the defect that is easy to leave in place while believing it fixed. · acceptance-sha256:a2120bdd34197aa1e77f41cc72443fca6614f9a7998b453c6dfc1fa3c975ba2e · covers:the odd value varies across seeds
- 2026-09-05 · e96504a* · mutant killed · exit 1 · `internal/curve/cell.go` · S3: let the relational fixture keep the constant common budget while only the odd one is drawn. The test fails on the common-value clause and contract 59 still passes its first assertion, which is the point: the signature would be half removed and every cell would still say retries = 3 for three of four blocks. · acceptance-sha256:a2120bdd34197aa1e77f41cc72443fca6614f9a7998b453c6dfc1fa3c975ba2e · covers:the odd value varies across seeds
- 2026-09-05 · e96504a* · mutant killed · exit 1 · `internal/curve/cell.go` · S3: let the draw reach the NAMED fixture. TestALegacyTrialIDStillRegenerates still passes — the id does not depend on the rendered text — but TestTheNamedSelectorIsUnchangedByTheRelationalOne and contract 59 both fail, because a named cell would render a drawn budget and stop being byte-identical to the fixture a committed reading was collected against. This is the risk the task names first and the reason 59 pairs the relational check with two named cells. · acceptance-sha256:a2120bdd34197aa1e77f41cc72443fca6614f9a7998b453c6dfc1fa3c975ba2e · covers:the named fixture is untouched
- 2026-09-05 · e96504a* · mutant killed · exit 1 · `internal/curve/cell.go` · S2: draw from a constant rather than from the trial seed. Every seed gets the same pair again, so TestTheOddRetryBudgetIsNotAFixedSignature fails on both clauses and contract 59 fails — the signature returns while the code still LOOKS drawn, which is the shape most likely to survive a reading of the diff. · acceptance-sha256:a2120bdd34197aa1e77f41cc72443fca6614f9a7998b453c6dfc1fa3c975ba2e · covers:the built binary draws it
- 2026-09-05 · 6de05a4* · mutant killed · exit 1 · `internal/curve/cell.go` · S2: draw the odd budget without excluding the common one, which is the rejection-free version a reviewer would write. TestTheOddRetryBudgetIsNotAFixedSignature fails and §59 fails: at seeds 13, 18, 23 and 24 the two draws collide and the cell has NO odd block, so the instruction describes a property the fixture lacks. Found by review of PR #94, which noted the first version of the test exercised seeds 1-8 and none of the colliding ones — the sweep is now 1-30 and the range is as much of the assertion as the clauses are. · acceptance-sha256:ace36100f2b6c7d2be3c6856e68118bd6fc47cc508f551f420fb65b2b2e62c1e · covers:the odd value varies across seeds
- 2026-09-05 · 6de05a4* · mutant killed · exit 1 · `internal/curve/cell.go` · S2: fix the COMMON budget and draw only the odd one. This is the half of the defect that looks fixed, and the first version of §59 passed it: comparing whole multisets, a fixed common with a varying odd still gives two different signatures. §59 now checks both values separately across four seeds and fails, as does the test. A constant common value identifies the odd block by elimination. · acceptance-sha256:ace36100f2b6c7d2be3c6856e68118bd6fc47cc508f551f420fb65b2b2e62c1e · covers:the odd value varies across seeds
- 2026-09-05 · 6de05a4* · mutant killed · exit 1 · `internal/curve/cell.go` · S3: let the draw reach the NAMED fixture. TestTheNamedFixtureMatchesItsGoldenBytes fails on digests taken from origin/main, and §59 fails on the named pair. TestALegacyTrialIDStillRegenerates does NOT fail — the id hashes parameters, so drifted bytes reuse it in silence, which is why the golden test exists and why the task no longer claims the id protects this. · acceptance-sha256:ace36100f2b6c7d2be3c6856e68118bd6fc47cc508f551f420fb65b2b2e62c1e · covers:the named fixture is untouched
- 2026-09-05 · 6de05a4* · mutant killed · exit 1 · `internal/curve/cell.go` · S4: draw from a constant rather than from the trial seed. Every seed gets the same pair, so the signature returns while the code still LOOKS drawn — the shape most likely to survive a reading of the diff. The test fails on both clauses and §59 fails on both budget sets. · acceptance-sha256:ace36100f2b6c7d2be3c6856e68118bd6fc47cc508f551f420fb65b2b2e62c1e · covers:the built binary draws it
- 2026-09-05 · 6de05a4* · mutant killed · exit 1 · `internal/curve/cell.go` · S2: draw from a package-level generator that ADVANCES instead of from the seed. It does not compile alone, so this row is the paired half of the mutation whose compiling form was run by hand: with a var statefulRNG added, TestTheDrawIsStableAcrossThePaddingFit fails because render is called repeatedly while the padding is fitted and the budgets move between steps. Nothing else notices: every budget 2..8 is one digit, so the fit still converges — on a different cell from the one measured. · acceptance-sha256:ace36100f2b6c7d2be3c6856e68118bd6fc47cc508f551f420fb65b2b2e62c1e · covers:the drawn pair is stable across the fit
- 2026-09-05 · 6de05a4* · mutant killed · exit 1 · `internal/curve/cell.go` · S2, and this one COMPILES, unlike the previous row for the same mechanism which failed at build and is therefore inconclusive rather than killed. Make the draw depend on the padding width, so it moves between the repeated render calls Generate makes while fitting the size cell. TestTheDrawIsStableAcrossThePaddingFit fails; nothing else does, because every budget 2..8 is one digit so the fit still converges — on a different cell from the one measured. · acceptance-sha256:ace36100f2b6c7d2be3c6856e68118bd6fc47cc508f551f420fb65b2b2e62c1e · covers:the drawn pair is stable across the fit

## Invariants

- Under `ByOddRetries` the odd budget takes more than one value across seeds, and the pair always differs.
- Under `ByName` every block renders the constant, and the fixture, served rendering and answer are byte-identical to what `origin/main` generates — pinned by golden digests, not inferred from the trial id.
- The drawn pair is stable across every call `render` makes while the padding is fitted.
- Every engine directory is byte-identical; `go.mod` declares exactly one requirement.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The draw reaches the named fixture | Low | **High** | Reading 2's 45 results were collected against those bytes. `TestTheNamedFixtureMatchesItsGoldenBytes` pins digests from `origin/main`'s binary and §59 checks the shape through the built one. **NOT** the trial id: it hashes parameters, so drifted bytes would reuse it silently |
| The draw is unstable across the padding-fit loop | Med | High | `retryPair` derives from the seed, not from a generator that advances, and `TestTheDrawIsStableAcrossThePaddingFit` renders one cell at five padding widths. An earlier version of this row claimed the fit loop would fail to converge and so catch it; that is false — every budget 2..8 is one digit, so the rendered length does not change and an unstable draw would converge on a different cell in silence |
| A drawn pair still leaks a signature | Low | Low | The values are small integers, so a client could still enumerate them. This removes the CONSTANT, not the alphabet, and the record says so rather than claiming more |

## Stop Condition

Stop if removing the signature needs the instruction to change. The instruction is the independent
variable's other half and T2 fixed it; a task that rewrites it is changing two things at once.

## Out of Scope

- Any change to the named fixture (permanent: boundary: T2 owns it, and a committed reading depends on it being byte-identical)
- Making the fixture resistant to an enumerating client (permanent: fact: the budgets are small integers and a client could try them all; citation: this task's Risks)
- Running a relational reading (deferred: docs/adr/BACKLOG.md — the relational-reading entry from T2)

## Verification Log
<!-- filled during execution -->
- 2026-09-05 · e96504a* · exit 0 · `set -o pipefail …` · acceptance-sha256:a2120bdd34197aa1e77f41cc72443fca6614f9a7998b453c6dfc1fa3c975ba2e · ms:25808
- 2026-09-05 · e96504a* · exit 0 · `set -o pipefail …` · acceptance-sha256:a2120bdd34197aa1e77f41cc72443fca6614f9a7998b453c6dfc1fa3c975ba2e · ms:25591
- 2026-09-05 · e96504a* · exit 0 · `set -o pipefail …` · acceptance-sha256:a2120bdd34197aa1e77f41cc72443fca6614f9a7998b453c6dfc1fa3c975ba2e · ms:25668
- 2026-09-05 · e96504a* · exit 0 · `set -o pipefail …` · acceptance-sha256:a2120bdd34197aa1e77f41cc72443fca6614f9a7998b453c6dfc1fa3c975ba2e · ms:25938
- 2026-09-05 · 6de05a4* · exit 0 · `set -o pipefail …` · acceptance-sha256:ace36100f2b6c7d2be3c6856e68118bd6fc47cc508f551f420fb65b2b2e62c1e · ms:26362
- 2026-09-05 · 6de05a4* · exit 0 · `set -o pipefail …` · acceptance-sha256:ace36100f2b6c7d2be3c6856e68118bd6fc47cc508f551f420fb65b2b2e62c1e · ms:25664
- 2026-09-05 · 6de05a4* · exit 0 · `set -o pipefail …` · acceptance-sha256:ace36100f2b6c7d2be3c6856e68118bd6fc47cc508f551f420fb65b2b2e62c1e · ms:26079
- 2026-09-05 · 6de05a4* · exit 0 · `set -o pipefail …` · acceptance-sha256:ace36100f2b6c7d2be3c6856e68118bd6fc47cc508f551f420fb65b2b2e62c1e · ms:26042
- 2026-09-05 · 6de05a4* · exit 0 · `set -o pipefail …` · acceptance-sha256:ace36100f2b6c7d2be3c6856e68118bd6fc47cc508f551f420fb65b2b2e62c1e · ms:25675
- 2026-09-05 · 6de05a4* · exit 0 · `set -o pipefail …` · acceptance-sha256:ace36100f2b6c7d2be3c6856e68118bd6fc47cc508f551f420fb65b2b2e62c1e · ms:25603
