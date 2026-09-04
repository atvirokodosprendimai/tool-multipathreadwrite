# Task ADR-020-T1: Generate a cell and score a result

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (one new package, one new binary, one contract row)
**Owner:** unassigned
**Produces:** the generator, the manifest shape, and the scorer
**Consumes:** `plan.Parse`, `apply.Apply`, `seen.Record`, `state.Dir` (all unchanged)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a wrong line is scored as a miss`, `results from another trial are refused`, `a refusal is not a localisation miss`, `each cell owns its ledger`

## Goal

A cell can be generated, handed to any client, and scored mechanically — and the scorer can be shown
to report a miss, because a scorer that has only ever seen correct answers proves nothing.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/curve/cell.go` | create | the generator: fixture tree, planted target, constant distractors, position stratum, manifest |
| `internal/curve/score.go` | create | the scorer: echo checks, whole-ledger seeding, apply, diff, three outcomes |
| `internal/curve/score_test.go` | create | **the ADR's Enforced-by**, plus the refusal and isolation properties |
| `cmd/curve/main.go` | create | the three verbs over the package — stdlib `flag`, so `go.mod` is unchanged |
| `scripts/contract.sh` | edit | §54 — drive the BUILT `curve` binary generate → score, for a right answer and a wrong one |

## Ordered Steps

1. [S1] Write the failing tests first (TDD red): a plan addressing the planted line scores a hit; a
   plan addressing a DIFFERENT real line scores a miss; results echoing another trial's id or
   another served size are refused; a plan that cannot parse is excluded from the denominator and
   counted separately. [proof: acceptance]
2. [S2] Generate the fixture with the target PLANTED, so the correct line is known by construction
   rather than judged. Distractors are near-identical — same shape, differing in the detail the
   instruction names — and their COUNT is held constant across size cells, with the remaining
   padding inert. This is the threat to validity the pre-registration names first: unique-string
   targets would measure string matching. [proof: mutation]
3. [S3] Carry the trial id and the MEASURED served bytes in the manifest, and refuse results that do
   not echo both. A manifest emitted at one size and results pasted back from another would
   otherwise score cleanly and mean nothing. [proof: mutation]
4. [S4] Score by APPLYING to a copy and diffing, not by comparing address strings. The changed line
   number is the measurement, and it is the parser and applier `mrw` itself uses. [proof: mutation]
5. [S5] Seed the ledger WHOLE before applying, so ADR-002's guard cannot convert a wrong address
   into a refusal and remove the worst misses from the primary denominator. [proof: mutation]
6. [S6] Report three outcomes. A refusal is excluded from the correct-address denominator and
   counted separately — never folded into either direction. [proof: acceptance]
7. [S7] Give each cell a fresh `XDG_STATE_HOME`, so no trial inherits another's ledger.
   [proof: mutation]
8. [S8] Add contract §54: build the `curve` binary, generate a cell, score a KNOWN-GOOD plan and a
   KNOWN-WRONG plan, and assert the two verdicts differ in the right direction. A row that scored
   only the good plan would pass with a scorer that always says hit. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
test -z "$(gofmt -l .)" \
  && go vet ./... \
  && go test ./internal/curve/ -v 2>&1 | tee /tmp/adr020-t1.out \
  && ! grep -qE 'no tests to run|no test files|^FAIL|^--- FAIL' /tmp/adr020-t1.out \
  && [ "$(grep -cE '^--- PASS: (TestTheScorerCountsAWrongLineAsAMiss|TestTheScorerRefusesResultsFromADifferentTrial|TestARefusedPlanIsNotCountedAsALocalisationMiss|TestEachCellGetsItsOwnLedger)\b' /tmp/adr020-t1.out)" = "4" ] \
  && grep -q '^# 54\.' scripts/contract.sh \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/mcp)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/mcp \
  && go test ./... \
  && ./scripts/contract.sh
```

Every clause was run BEFORE this fence was written and returned **zero hits**: the four test names,
`# 54.`, and the Go identifiers `Manifest` and `ScoreTrial`. **§54**: 53 is the highest on `main`,
confirmed by sorting the section numbers numerically rather than reading the last one in file order
— they are not in order in `contract.sh`, and taking the tail would have said something else.

**`internal/mcp` is in the go/no-go clauses here** and was not in ADR-017's or ADR-018's. This record
measures `MaxResultChars`; the moment it edits the file that declares it, the instrument and the
thing being measured have merged.

⚠ **THE UNIT TESTS ALONE CANNOT PROVE THIS.** A scorer can be correct and reachable from nothing, and
`generate`, `score` and `tally` can each work while the manifest they exchange does not round-trip
through the filesystem. §54 is the row that binds, because it drives the verbs of the BUILT binary in
sequence with a good plan and a wrong one.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestTheScorerCountsAWrongLineAsAMiss` | `internal/curve/score_test.go` | **the ADR's Enforced-by** — a plan that parses, applies and changes the wrong line is a miss, not a pass | — | S1, S2, S4, S5 |
| `TestTheScorerRefusesResultsFromADifferentTrial` | `internal/curve/score_test.go` | S3 — a mismatched trial id or served size is refused, not scored | — | S1, S3 |
| `TestARefusedPlanIsNotCountedAsALocalisationMiss` | `internal/curve/score_test.go` | S6 — the third outcome stays out of the primary denominator | — | S1, S6 |
| `TestEachCellGetsItsOwnLedger` | `internal/curve/score_test.go` | S7 — no trial inherits another's state | — | S1, S7 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the four tests above |
| 2 — something selects it | `cmd/curve` is the only caller; §54 drives it through the real binary |
| 3 — the caller can discover it | `curve -h` names all three verbs, and the ADR states the protocol |
| 4 — it is used | §54 runs a complete generate→score cycle. The first READING is a Follow-up and the record says so rather than implying data exists |

## Mutation Log
<!-- filled during execution -->
- 2026-09-04 · 2ab8d9a* · mutant killed · exit 1 · `internal/curve/score.go` · S4: score every applied plan as a hit — `if true {` in place of the changed-set check. TestTheScorerCountsAWrongLineAsAMiss fails on the distractor plan: a plan that parsed, applied cleanly and changed the WRONG line reported hit. This is the scorer the record exists to refuse — one that has only ever been shown correct answers · acceptance-sha256:de86ec445cbbd3d71c02bd58bf4c984b24725f97de8315d5438bc5090ee3bfa1
- 2026-09-04 · 2ab8d9a* · mutant killed · exit 1 · `internal/curve/score.go` · S3: drop the trial-id echo check. TestTheScorerRefusesResultsFromADifferentTrial fails: a result naming "someone-else" was scored against this trial's answer instead of refused · acceptance-sha256:de86ec445cbbd3d71c02bd58bf4c984b24725f97de8315d5438bc5090ee3bfa1
- 2026-09-04 · 2ab8d9a* · mutant killed · exit 1 · `internal/curve/score.go` · S5: seed the ledger with ONE served line instead of the whole file. TestTheScorerCountsAWrongLineAsAMiss fails: the plan at the planted line came back refused_apply — ADR-002's guard turned a hit into a refusal, which is the leak from the primary denominator that Decision 4 exists to stop · acceptance-sha256:de86ec445cbbd3d71c02bd58bf4c984b24725f97de8315d5438bc5090ee3bfa1
- 2026-09-04 · 2ab8d9a* · mutant killed · exit 1 · `internal/curve/cell.go` · S2: make the inert padding emit the distractor line. TestDistractorCountDoesNotVaryWithSize fails: the 40,000-byte cell carried hundreds of candidates against the 4,000-byte cell's five — size and distractor count moving together, the confound the pre-registration names · acceptance-sha256:de86ec445cbbd3d71c02bd58bf4c984b24725f97de8315d5438bc5090ee3bfa1
- 2026-09-04 · 2ab8d9a* · mutant killed · exit 1 · `internal/curve/cell.go` · S7: put every cell's state home at one shared temp path. TestEachCellGetsItsOwnLedger fails: two cells named the same directory, outside either cell · acceptance-sha256:de86ec445cbbd3d71c02bd58bf4c984b24725f97de8315d5438bc5090ee3bfa1
- 2026-09-04 · 2ab8d9a* · mutant killed · exit 1 · `internal/curve/score.go` · S6: count a refused plan in the cell's N. TestARefusedPlanIsNotCountedAsALocalisationMiss fails: N became 3 and the rate moved with the refusal. S6 is [proof: acceptance] in this task; the mutant was cheap and the denominator rule is the one a reader would question, so it is logged · acceptance-sha256:de86ec445cbbd3d71c02bd58bf4c984b24725f97de8315d5438bc5090ee3bfa1

## Invariants

- A plan addressing the planted line scores a hit; a plan addressing another real line scores a miss.
- Results that do not echo the trial id and the served size are refused.
- A refused plan is excluded from the correct-address denominator and reported separately.
- The ledger is seeded whole, so a wrong address cannot arrive as a refusal.
- Each cell has its own `XDG_STATE_HOME`.
- The distractor count does not vary with the size cell.
- No engine or `internal/mcp` change; `go.mod` declares exactly one requirement.

## Risks

- The scorer is only ever fed correct plans and reports 100%. The Enforced-by feeds it a wrong one.
- The verbs work in-process and the manifest does not survive the filesystem. §54 crosses it.
- Padding becomes distractors and a second variable rides along. The count is asserted constant.

## Stop Condition

Stop if scoring needs `internal/plan`, `internal/apply` or `internal/seen` to learn anything new. The
harness's whole claim is that it uses the engine callers use; the moment it needs a variant, it is
measuring something else and the record should say so rather than the branch quietly widening.

## Out of Scope

- The first reading (deferred: parent ADR, Follow-ups — pre-registered in docs/adr/BACKLOG.md)
- Any model client or network call (permanent: boundary: parent ADR, Decision 1)
- Changing `MaxResultChars` (permanent: boundary: ADR-011-T3 owns it)
- Distractor density as its own curve (permanent: boundary: parent ADR, Out of Scope)

## Verification Log
<!-- filled during execution -->
- 2026-09-04 · 2ab8d9a* · exit 0 · `set -o pipefail …` · acceptance-sha256:de86ec445cbbd3d71c02bd58bf4c984b24725f97de8315d5438bc5090ee3bfa1 · ms:26211
