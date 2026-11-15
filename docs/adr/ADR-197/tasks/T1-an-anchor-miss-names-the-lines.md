# Task ADR-197-T1: An anchor miss names the lines that hold the anchor

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** `anchorMiss`'s closing clause; `TestAnAnchorMissNamesTheLinesThatHoldIt`; contract §337
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the lines holding the anchor are named`, `no line holding it is said`, `the list stops at five`, `the built binary names the line`, `README and AGENTS.md name the clause`, `the packages the task does not own are unchanged`

## Goal

Decision 1 of the record.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/apply/anchor197_test.go` | new | the red test |
| `internal/apply/apply.go` | edit | `anchorMiss` (`apply.go:2655`) ends with where the anchor is |
| `scripts/contract.sh` | edit | §337, through the built binary |
| `README.md`, `AGENTS.md` | edit | one sentence where `anchor=` is described |

## Ordered Steps

1. [S1] Write the failing test `TestAnAnchorMissNamesTheLinesThatHoldIt` and see it RED. Cases, through `Apply` with every line read:
   - the anchor is in one other line: the reason ends `; the anchor is in line 4`;
   - in three other lines: `lines 4, 7, 9`;
   - in seven: the first five, then `and 2 more`;
   - in no line: `; no line of f.go holds it`;
   - the #589 case (the line holds the anchor without its double quotes, no line holds it as written): the quote advice is there, and neither `no line of` nor `the anchor is in` is;
   - the anchor is in the addressed line's neighbour only as part of a longer word: it is named, since the test is the one the anchor itself gets, a substring;
   - an anchor that matches: no refusal, as now.
2. [S2] `anchorMiss` reports what its scan finds. Mutants, each killed:
   - the clause dropped, so the one-line case goes red;
   - the cap of five removed, so the seven-line case goes red;
   - `no line of` printed when lines do hold it, so the three-line case goes red.
   [proof: mutation]
3. [S3] Contract §337, through the built binary: a two-line replace whose anchor sits one line below the range exits 1, writes nothing and names that line; the pair: the same plan re-addressed to the named line exits 0. [proof: acceptance]
4. [S4] README and AGENTS.md say so. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(go test ./internal/apply/ -count=1 -v -run '^TestAnAnchorMissNamesTheLinesThatHoldIt$' 2>&1) \
  && grep -qE '^--- PASS: TestAnAnchorMissNamesTheLinesThatHoldIt \(' <<<"$out" \
  && c=$(./scripts/contract.sh 2>&1) \
  && grep -q '^# 337\. ' scripts/contract.sh \
  && grep -qF 'PASS  an anchor miss names the line that holds the anchor and writes nothing' <<<"$c" \
  && grep -qF 'PASS  the pair: the plan re-addressed to that line applies' <<<"$c" \
  && grep -qF 'the anchor is in line' AGENTS.md \
  && grep -qF 'the anchor is in line' README.md \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- go.mod go.sum internal/plan internal/read internal/check internal/state internal/seen internal/writer \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/plan internal/read internal/check internal/state internal/seen internal/writer)" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAnAnchorMissNamesTheLinesThatHoldIt` | `internal/apply/anchor197_test.go` | the lines holding the anchor are named, capped at five, or none is said; the quote advice survives | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test |
| 2 — something selects it | every `anchor=` hunk, CLI and `mrw_write`, through `Apply` |
| 3 — the caller can discover it | it is in the refusal itself; README and AGENTS.md |
| 4 — it is used | no telemetry (ADR-009); the evidence for building it is 140 anchor misses in 116 transcripts, 104 followed by a read (record, Context) |

## Mutation Log
- 2026-11-15 · 663b047e* · mutant killed · exit 1 · `internal/apply/apply.go` · the clause dropped for one holding line, so the one-line case must go red · acceptance-sha256:d28930553aa0afe99b861d918397d316730b9875df91341a00cf798f6178d594 · covers:the lines holding the anchor are named
- 2026-11-15 · 663b047e* · mutant killed · exit 1 · `internal/apply/apply.go` · the cap of five lifted, so seven holding lines are all named and the seven-line case must go red · acceptance-sha256:d28930553aa0afe99b861d918397d316730b9875df91341a00cf798f6178d594 · covers:the list stops at five
- 2026-11-15 · 663b047e* · mutant killed · exit 1 · `internal/apply/apply.go` · no line of printed when lines do hold the anchor, so the three-line case must go red · acceptance-sha256:d28930553aa0afe99b861d918397d316730b9875df91341a00cf798f6178d594 · covers:no line holding it is said
- 2026-11-15 · 663b047e* · mutant killed · exit 1 · `scripts/contract.sh` · the row expects another line number, so the built binary naming line 3 must fail the row · acceptance-sha256:d28930553aa0afe99b861d918397d316730b9875df91341a00cf798f6178d594 · covers:the built binary names the line
- 2026-11-15 · 663b047e* · mutant killed · exit 1 · `AGENTS.md` · the sentence loses the clause, so the AGENTS.md grep must go red · acceptance-sha256:d28930553aa0afe99b861d918397d316730b9875df91341a00cf798f6178d594 · covers:README and AGENTS.md name the clause
- 2026-11-15 · 663b047e* · mutant killed · exit 1 · `internal/plan/plan.go` · a package the task does not own is changed, so the byte-identical clause must go red · acceptance-sha256:d28930553aa0afe99b861d918397d316730b9875df91341a00cf798f6178d594 · covers:the packages the task does not own are unchanged

## Verification Log
- 2026-11-15 · 663b047e* · exit 1 · `set -o pipefail …` · acceptance-sha256:d28930553aa0afe99b861d918397d316730b9875df91341a00cf798f6178d594 · ms:547 · test-lock-sha256:d41b0403233945480b7274e1e176802e8d5d218babc523f098c368d21359bcf5 · test-lock-file:docs/adr/locks/d41b0403233945480b7274e1e176802e8d5d218babc523f098c368d21359bcf5.lock
  ```
  ```
- 2026-11-15 · 663b047e* · exit 0 · `set -o pipefail …` · acceptance-sha256:d28930553aa0afe99b861d918397d316730b9875df91341a00cf798f6178d594 · ms:98179
- 2026-11-15 · 663b047e* · exit 0 · `set -o pipefail …` · acceptance-sha256:d28930553aa0afe99b861d918397d316730b9875df91341a00cf798f6178d594 · ms:108245
- 2026-11-15 · 663b047e* · exit 0 · `set -o pipefail …` · acceptance-sha256:d28930553aa0afe99b861d918397d316730b9875df91341a00cf798f6178d594 · ms:91977
- 2026-11-15 · 663b047e* · exit 0 · `set -o pipefail …` · acceptance-sha256:d28930553aa0afe99b861d918397d316730b9875df91341a00cf798f6178d594 · ms:89694
- 2026-11-15 · 663b047e* · exit 0 · `set -o pipefail …` · acceptance-sha256:d28930553aa0afe99b861d918397d316730b9875df91341a00cf798f6178d594 · ms:89909
- 2026-11-15 · 663b047e* · exit 0 · `set -o pipefail …` · acceptance-sha256:d28930553aa0afe99b861d918397d316730b9875df91341a00cf798f6178d594 · ms:90071
- 2026-11-15 · 663b047e* · exit 0 · `set -o pipefail …` · acceptance-sha256:d28930553aa0afe99b861d918397d316730b9875df91341a00cf798f6178d594 · ms:86568

## Invariants

- An anchor that matches is unchanged, and so is every other refusal.
- The refusal quotes no line it did not already quote.

## Risks

- None beyond the record's. The assertions found on 663b047e are substrings; search again before editing.

## Stop Condition

The fence exits 0. Stop and ask Zy if any existing assertion needs the old reason to END where it ends now.

## Out of Scope

- Everything the record's Out of Scope names (deferred: docs/adr/BACKLOG.md)
