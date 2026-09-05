# Task ADR-020-T4: A served window that does not begin at line one

**Depends-on:** T3
**Covers:** none — no spec
**Estimated scope:** XS (one field, one range on the serve, one test, one contract row)
**Owner:** unassigned
**Produces:** a cell whose served rendering starts at a chosen line, so a row count and a line number diverge
**Consumes:** `Generate`, `serve`, `read.Range` (unchanged), `Params` (T2/T3)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the window starts where asked`, `the target is inside the window`, `a whole-file cell is byte-identical`, `the built binary serves a window`

## Goal

The generator can serve a fixture from a chosen line rather than from line 1, so the target's ROW in
the served rendering and its LINE NUMBER plus two are different integers. That is the one thing every
reading so far has lacked, and it is what makes the `target+2` mechanism decidable.

## Why this task exists

Across the 135 read-arm trials of readings 2, 3 and 4 there are 13 misses and all 13 addressed the
line exactly two below the target. Every cell serves `@@ 1-N`, and mrw's served rendering opens with
two rows that carry no line number — the `==>` header and the `@@` range. So in every trial so far,
*row of the target in the rendering* and *target line number + 2* are the same integer, and no
reading can tell "the client counted rows" from "the client added two for another reason".

Reading 2 named this experiment and deferred it: at 3 misses in 45 it could not be powered. Reading 4
found a client that misses 7 in 15 at 200 KB. **With a window served from, say, line 500, the two
accounts differ by 498 lines and one trial that misses tells them apart.** This task builds the cell;
running the trials is a reading and a budget decision, as always in this record.

If the row-count account holds, the tool's own output induces the silent wrong write the tool exists
to prevent, and the fix belongs in `internal/read` under its own record — not here.

**Superseded 2026-09-05 by reading 5** (`docs/curve/reading-05-result.md`): the row-count account
held (every miss at `target − 117` from line 120), but the sentence above assumed the only rows a
client could count were mrw's. The read arm delivers `served.txt` through the client's own file
reader, which lays a second gutter beside mrw's, and the transcript suggests the client took that one.
Whether mrw's rendering induces the count when no other gutter is present was decided by reading 8
(`docs/curve/reading-08-result.md`, 2026-09-05): it does not — 15 of 15 through a tool result — so
no `internal/read` record opens; the BACKLOG entry is closed.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/curve/cell.go` | edit | `Params.ServeFrom`; `serve` passes a `read.Range{Start: from}`; `check` and `Generate` refuse a window that excludes the target |
| `internal/curve/cell_test.go` | edit | **the task's Enforced-by** — the served rendering's `@@` header starts at the requested line, the target is inside it, and `ServeFrom: 0` is byte-identical to before |
| `cmd/curve/main.go` | edit | `-from` on `generate` |
| `scripts/contract.sh` | edit | §60 — the built binary serves a window and refuses one that excludes the target |

## Ordered Steps

1. [S1] Write the failing test first (TDD red): with `ServeFrom: 400` on a 200,000-byte cell, the
   served rendering's `@@` line begins at 400, the answer's line is ≥ 400, and the served bytes are
   fewer than the whole-file cell's; with `ServeFrom: 0` the cell is byte-identical to T3's golden
   digest. [proof: acceptance]
2. [S2] Add `Params.ServeFrom` (zero means the whole file) and pass `read.Range{Start: ServeFrom}` in
   `serve`. The fit loop measures the WINDOW's bytes, because that is what the client is served.
   [proof: mutation]
3. [S3] After the fit, refuse a window whose start is past the target's line: a cell whose answer is
   not in what the client sees is unanswerable, not hard. [proof: mutation]
4. [S4] `-from` on `curve generate`, and contract §60: the built binary serves a window whose header
   starts where asked, paired with a `-from` past the target that exits non-zero. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
test -z "$(gofmt -l .)" \
  && go vet ./... \
  && go test ./internal/curve/ -v 2>&1 | tee /tmp/adr020-t4.out \
  && ! grep -qE 'no tests to run|no test files|^FAIL|^--- FAIL' /tmp/adr020-t4.out \
  && [ "$(grep -cE '^--- PASS: (TestAServedWindowNeedNotBeginAtLineOne)\b' /tmp/adr020-t4.out)" = "1" ] \
  && [ "$(grep -cE '^--- PASS: (TestTheNamedFixtureMatchesItsGoldenBytes)\b' /tmp/adr020-t4.out)" = "1" ] \
  && grep -q '^# 60\. ' scripts/contract.sh \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/mcp \
  && go test ./... \
  && ./scripts/contract.sh
```

Every clause was run BEFORE this fence was written and returned **zero hits**: the test name, `# 60. `
and the Go identifier `ServeFrom`. **§60**: 59 is the highest on `main` by the tightened recipe.

The golden-digest clause is deliberate: the cheapest way to get a window wrong is to change what a
whole-file cell serves, and T3's golden test is what catches that. `internal/read` is in the go/no-go
clauses: this task USES a range the read engine already offers and must not touch the engine.

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAServedWindowNeedNotBeginAtLineOne` | `internal/curve/cell_test.go` | **the task's Enforced-by** — the `@@` header starts where asked, the target is inside, the window is smaller than the file, and a window past the target is refused | — | S1, S2, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the test above |
| 2 — something selects it | `-from` on `curve generate`; §60 drives it through the built binary |
| 3 — the caller can discover it | `curve generate -h` lists the flag |
| 4 — it is used | not yet: the reading that uses it is a Follow-up. The evidence for building it is 13 of 13 misses at one offset and no cell in the corpus that could tell why |

## Mutation Log
<!-- filled during execution -->
- 2026-09-05 · dbc1fb9* · mutant killed · exit 1 · `internal/curve/cell.go` · S2: accept -from, print its header text, and serve from line 1 anyway. TestAServedWindowNeedNotBeginAtLineOne fails on the range header and contract 60 fails on the same — the exact shape that would leave every other test green while the discriminating reading measured a window that was never there. · acceptance-sha256:447a404b6802467de7d96158e0259d7d9445e71c86a77c6e2ad5dfde59eec85e · covers:the window starts where asked
- 2026-09-05 · dbc1fb9* · mutant survived · exit 0 · `internal/curve/cell.go` · S3: never refuse a window past the target. SURVIVED: the fence passed with the post-fit check deleted, because no fixture reached that branch — the only past-the-target case used line 100000, which the fit loop refuses first. (This row's prose was written before the run as the expected outcome and originally claimed the test failed; corrected by hand after review of PR #100 to say what happened, with verdict and digest untouched.) · acceptance-sha256:447a404b6802467de7d96158e0259d7d9445e71c86a77c6e2ad5dfde59eec85e · covers:the target is inside the window
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-09-05 · dbc1fb9* · mutant killed · exit 1 · `internal/curve/cell.go` · S2, the other half: ignore ServeFrom in the fit loop and size the WHOLE file. The window is then served afterwards over a fixture sized for the whole file, so the served bytes fall short of the cell and the test fails on the byte-count clause: the fit must measure what the client is served, not what exists. · acceptance-sha256:447a404b6802467de7d96158e0259d7d9445e71c86a77c6e2ad5dfde59eec85e · covers:the window starts where asked
- 2026-09-05 · dbc1fb9* · mutant killed · exit 1 · `internal/curve/cell.go` · S3, re-run after the SURVIVED row above. The first run survived because no fixture reached the post-fit branch: the only past-the-target case used line 100000, which the fit loop refuses first. A window from 200 over an early target at ~76 exists in full and holds no answer; that case is now in the test and in contract 60, and this mutant fails both. The survived row is kept — it is the evidence that the branch was unreached, which testing.md says is the first thing a survivor means. · acceptance-sha256:447a404b6802467de7d96158e0259d7d9445e71c86a77c6e2ad5dfde59eec85e · covers:the target is inside the window
- 2026-09-05 · 8ab21c4 · mutant killed · exit 1 · `internal/curve/cell.go` · S2: drop the window from the trial id. TestAServedWindowNeedNotBeginAtLineOne fails on the twin-id clause and contract 60 fails on the same — a windowed cell and its whole-file twin become one trial to the scorer, the defect both reviews of PR #100 found. Logged after the fix at the reviewer's note, so the log says what the fence proves. · acceptance-sha256:447a404b6802467de7d96158e0259d7d9445e71c86a77c6e2ad5dfde59eec85e · covers:the window starts where asked

## Invariants

- `ServeFrom: 0` serves the whole file by the path that existed before T4, and the three fixtures T3 pinned by golden digest are byte-identical — which proves that path unchanged for those three, not for every cell.
- With `ServeFrom: N > 0` the served rendering's `@@` header begins at N and the answer's line is ≥ N.
- A window that excludes the target is refused at generate time, never scored.
- `internal/read` and every other engine directory are byte-identical.

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| The window changes what a whole-file cell serves | Low | **High** | `ServeFrom: 0` is the zero value and takes the same path as before; the golden test is in the fence |
| The client is served a window that does not contain the answer | Med | High | Refused after the fit, when the target's line is known; §60 pairs the good case with this one |
| The fit loop targets the wrong byte count | Med | Med | It measures what `serve` returns, and `serve` now returns the window; a test asserts the window is smaller than the whole-file cell |

## Stop Condition

Stop if serving a window needs any change under `internal/read`. The range exists; if it does not do
what this task needs, that is a finding about the engine and its own record, not a fixture change.

## Out of Scope

- Running the discriminating reading (permanent: fact: it was run as reading 5 on 2026-09-05 with the cell this task built, and the BACKLOG offset-window entry it was deferred to carries the receipt; citation: docs/curve/reading-05-result.md)
- Fixing the read format if the row-count account holds (permanent: fact: the account held and the fix was not needed — reading 8 showed the count was the harness read arm's, and the BACKLOG read-format entry closed with no engine change; citation: docs/curve/reading-08-result.md)
- Keying the retry pair on the whole tuple, issue #97 (deferred: issue #97 — not needed for a one-cell-per-client reading)

## Verification Log
<!-- filled during execution -->
- 2026-09-05 · dbc1fb9* · exit 0 · `set -o pipefail …` · acceptance-sha256:447a404b6802467de7d96158e0259d7d9445e71c86a77c6e2ad5dfde59eec85e · ms:34893
- 2026-09-05 · dbc1fb9* · exit 0 · `set -o pipefail …` · acceptance-sha256:447a404b6802467de7d96158e0259d7d9445e71c86a77c6e2ad5dfde59eec85e · ms:29377
- 2026-09-05 · dbc1fb9* · exit 0 · `set -o pipefail …` · acceptance-sha256:447a404b6802467de7d96158e0259d7d9445e71c86a77c6e2ad5dfde59eec85e · ms:25935
- 2026-09-05 · dbc1fb9* · exit 0 · `set -o pipefail …` · acceptance-sha256:447a404b6802467de7d96158e0259d7d9445e71c86a77c6e2ad5dfde59eec85e · ms:28633
- 2026-09-05 · 8ab21c4 · exit 0 · `set -o pipefail …` · acceptance-sha256:447a404b6802467de7d96158e0259d7d9445e71c86a77c6e2ad5dfde59eec85e · ms:50486
