# Task ADR-020-T3: The odd retry budget is drawn, not fixed

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** XS (one draw in the generator, one test, one contract assertion)
**Owner:** unassigned
**Produces:** a relational fixture with no constant signature
**Consumes:** `render`, `Selector` (T2, otherwise unchanged)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the odd value varies across seeds`, `the named fixture is untouched`, `the built binary draws it`

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
  && [ "$(grep -cE '^--- PASS: (TestALegacyTrialIDStillRegenerates)\b' /tmp/adr020-t3.out)" = "1" ] \
  && grep -q '^# 59\.' scripts/contract.sh \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/mcp \
  && go test ./... \
  && ./scripts/contract.sh
```

Every clause was run BEFORE this fence was written and returned **zero hits**: the test name, `# 59.`
and the Go identifier `retryPair`. **§59**: 58 is the highest on `main`.

The legacy-id clause is in the fence deliberately. This task changes the renderer, and the cheapest
way to get a drawn value wrong is to let it reach the NAMED fixture too — which would change ids
recorded in a committed reading. T2's regression test is the one that catches it, so this fence names
it rather than trusting the package-wide clause to notice.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheOddRetryBudgetIsNotAFixedSignature` | `internal/curve/cell_test.go` | **the task's Enforced-by** — across seeds the odd budget takes more than one value, the pair always differs, and a named cell still renders the constant in every block | — | S1, S2, S3 |

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

## Invariants

- Under `ByOddRetries` the odd budget takes more than one value across seeds, and the pair always differs.
- Under `ByName` every block renders the constant, so named cells stay byte-identical and their recorded trial ids still regenerate.
- Every engine directory is byte-identical; `go.mod` declares exactly one requirement.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The draw reaches the named fixture | Low | **High** | It would change ids in a committed reading. The fence names `TestALegacyTrialIDStillRegenerates` explicitly, and §59 pairs the relational check with two named cells that must be identical |
| The draw is unstable across the padding-fit loop | Med | High | `render` is called repeatedly while fitting; `retryPair` derives from the seed, not from a generator that advances, and the fit loop would otherwise never converge — which the existing tests would show as a generate error |
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
