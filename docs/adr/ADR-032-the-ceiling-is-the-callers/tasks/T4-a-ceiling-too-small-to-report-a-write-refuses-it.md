# Task ADR-032-T4: A ceiling too small to report a write refuses it, and every answer is bounded

**Depends-on:** T1, T2, T3
**Covers:** none — no spec
**Estimated scope:** M (a pre-apply guard, a funnel, and the four branches the first cut's tests never reached)
**Owner:** Zy
**Produces:** the pre-apply refusal, and one ceiling postcondition for every tool result
**Consumes:** the configured budget (T1), the bounded write receipt (T2)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a write refused rather than applied when its verdict cannot be reported`, `every tool result inside the advertised ceiling, refusals included`, `a failure surviving the second elision stage`

## Goal

Stop this record's own implementation from telling a caller that nothing was written after it wrote,
and make the ceiling it advertises true of every answer rather than of the large ones.

## Context this task exists for

The Codex review of PR #135 found two HIGH defects in T1/T2's implementation. Both were reproduced
on the built binary at `e8c1a29` before anything was changed.

**1. A write applied and reported that it had not.** `boundedReceipt`'s terminal branch said
"nothing was written" unconditionally, reasoning that a receipt too large to shorten must be all
failures and that ADR-001 therefore wrote nothing. A small ceiling breaks that reasoning:

    $ mrw mcp --max-result-chars 0     # line 1 already served
    BEFORE: alpha
    isError: True
    TEXT   : 0 of 1 hunk(s) failed and nothing was written. ...
    AFTER : MUTATED

The file changed, the ledger recorded it, and the receipt denied both. That is a false statement
about the filesystem — a worse defect than the truncation this record was written to fix, and
exactly the class the tool exists to refuse.

**2. The advertised ceiling was not enforced on every path.** `errorResult` carried no size check at
all, so at a small ceiling the REFUSALS exceeded the number `_meta` advertises. T1's Invariants claim
"the advertised value IS the enforced value"; that was false for every refusal, the grep no-match
answer, early write parse failures, and `matchIndex`'s minimum-one exception.

⚠ **Both defects were reachable through T1's own fence and §70.** The test is named
`TestTheAdvertisedCeilingBoundsEveryAnswer` and exercised one read and one DRY-RUN write at two
budgets. It never ran a real write, never ran a small budget, and never touched a refusal — a
universal name over a two-case fixture, which is the failure `testing.md` names first.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/tools.go` | edit | `callTool` gains one postcondition, `withinCeiling`, instead of a size check per return site — enumerating them is how four were missed. `writeTool` refuses before `apply.Apply` when the ceiling cannot carry a minimal truthful post-apply sentence. `boundedReceipt`'s terminal branch tells the truth when `res.Applied`, rather than relying on the new guard elsewhere staying correct. |
| `internal/mcp/limit_test.go` | edit | Four tests for the four unreached branches. |
| `scripts/contract.sh` | edit | §70 counts UTF-8 BYTES, not decoded characters, and drives a real write at a small ceiling. |
| `docs/adr/ADR-032-…md`, `docs/adr/…/tasks/T2-…md`, `README.md`, `AGENTS.md` | edit | Four claims the review found overstated or false. ⚠ An earlier version of this row named `internal/mcp/instructions.go`, which the first commit did NOT touch, and omitted T2, which it did — a row that describes a different change than the one it sits in. Corrected in the follow-up commit, which does edit the instructions. |

## Ordered Steps

1. [S1] Write `TestASmallCeilingRefusesTheWriteBeforeApplying` and confirm it is RED — the fixture asserts the FILE, not the message, because a fix that only corrected the wording would leave a write applying under a ceiling that cannot report it. [proof: mutation]
2. [S2] Refuse before `apply.Apply` when a minimal post-apply receipt would not fit, as a JSON-RPC error: it carries no `result` member, so it is outside the ceiling it is reporting on. [proof: mutation]
3. [S3] Make `boundedReceipt`'s terminal branch honest for an applied plan, independently of S2. [proof: mutation]
4. [S4] Add `withinCeiling` at `callTool` and write `TestEveryAnswerFitsIncludingTheRefusals`. [proof: mutation]
5. [S5] Write `TestTheCeilingNeverShrinksAServedRead`, which pins the ordering S4 rests on: a served read is refused for size BEFORE its ledger record is written, so the funnel never rewrites an answer that licensed something. [proof: mutation]
6. [S6] Write `TestTheSecondStageElisionDropsFileRecords` — T2's fixture had one file, so `Files = nil` never executed. [proof: mutation]
7. [S7] §70 counts `len(substring.encode("utf-8"))` rather than `len()` on a decoded string, and drives a real write at a small ceiling. [proof: acceptance]
8. [S8] Correct the four documentation claims. [proof: human: read each against the implementation it describes, not against neighbouring prose]
9. [S9] Run every gate. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/mcp/ -count=1 -v \
  -run 'TestASmallCeilingRefusesTheWriteBeforeApplying|TestEveryAnswerFitsIncludingTheRefusals|TestTheCeilingNeverShrinksAServedRead|TestTheSecondStageElisionDropsFileRecords|TestAPartialApplicationIsNotReportedAsNothingWritten|TestTheWriteFloorIsAFloor' 2>&1 | tee /tmp/adr032-t4.out \
  && grep -q '^--- PASS: TestASmallCeilingRefusesTheWriteBeforeApplying' /tmp/adr032-t4.out \
  && grep -q '^--- PASS: TestEveryAnswerFitsIncludingTheRefusals' /tmp/adr032-t4.out \
  && grep -q '^--- PASS: TestTheCeilingNeverShrinksAServedRead' /tmp/adr032-t4.out \
  && grep -q '^--- PASS: TestTheSecondStageElisionDropsFileRecords' /tmp/adr032-t4.out \
  && grep -q '^--- PASS: TestAPartialApplicationIsNotReportedAsNothingWritten' /tmp/adr032-t4.out \
  && grep -q '^--- PASS: TestTheWriteFloorIsAFloor' /tmp/adr032-t4.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr032-t4.out \
  && go test ./... -count=1 \
  && go test -race ./... -count=1 \
  && ./scripts/contract.sh 2>&1 | tee /tmp/adr032-t4c.out \
  && grep -q '^contract holds$' /tmp/adr032-t4c.out \
  && ! grep -qE '^ +FAIL ' /tmp/adr032-t4c.out \
  && [ -z "$(gofmt -l .)" ] \
  && go vet ./...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestASmallCeilingRefusesTheWriteBeforeApplying` | `internal/mcp/limit_test.go` | At budgets 0, 1 and 64 a licensed write leaves the file byte-identical and answers with a JSON-RPC error carrying no tool result | — | S1, S2, S3 |
| `TestEveryAnswerFitsIncludingTheRefusals` | `internal/mcp/limit_test.go` | Five refusal shapes — oversized read, bad spec, exclude without grep, empty grep, unparseable plan — all fit the ceiling in force | — | S4 |
| `TestTheCeilingNeverShrinksAServedRead` | `internal/mcp/limit_test.go` | A read refused for encoded size licenses no write, so the funnel never discards content the ledger recorded | — | S5 |
| `TestTheSecondStageElisionDropsFileRecords` | `internal/mcp/limit_test.go` | With 400 files and 2 failures at a 3,000-byte ceiling, the file records go, the elision says so, and both failures survive | — | S6 |
| `TestAPartialApplicationIsNotReportedAsNothingWritten` | `internal/mcp/limit_test.go` | A result with `Applied: false` and one file `Written: true` is reported as PARTIALLY APPLIED with the file counted, never as "nothing was written" | — | S3 |
| `TestTheWriteFloorIsAFloor` | `internal/mcp/limit_test.go` | At ten ceilings, the floor bounds the real message at every count width including `math.MaxInt`, and the write is refused exactly when the floor exceeds the ceiling | — | S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | The four tests above |
| 2 — something selects it | `withinCeiling` is on the single `callTool` funnel every tools/call passes through; the pre-apply guard is on the only path to `apply.Apply` from MCP |
| 3 — the caller can discover it | The refusal names the ceiling in force and the minimum that would work |
| 4 — it is used | §70 drives the built server at a small ceiling; no telemetry, per ADR-009 |

## Mutation Log

- 2026-09-07 · e8c1a29* · mutant killed · exit 1 · `internal/mcp/tools.go` · the pre-apply guard is gone, so a write applies under a ceiling that cannot report it and the caller is told nothing happened — the defect reproduced on the built binary at e8c1a29 · acceptance-sha256:b372ba8dfb64d297b8ccf0fd27d7fd7fc2499fab220ec6457faa367a879bc240 · covers:a write refused rather than applied when its verdict cannot be reported
- 2026-09-07 · e8c1a29* · mutant survived · exit 0 · `internal/mcp/tools.go` · the funnel stops bounding anything, so a refusal larger than the advertised ceiling goes out — the state ADR-032 shipped in, where errorResult carried no size check at all · acceptance-sha256:b372ba8dfb64d297b8ccf0fd27d7fd7fc2499fab220ec6457faa367a879bc240 · covers:every tool result inside the advertised ceiling, refusals included
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-09-07 · e8c1a29* · mutant killed · exit 1 · `internal/mcp/tools.go` · the second elision stage stops dropping file records, so a receipt with hundreds of files never fits and the caller gets the terminal refusal instead of a usable verdict · acceptance-sha256:b372ba8dfb64d297b8ccf0fd27d7fd7fc2499fab220ec6457faa367a879bc240 · covers:a failure surviving the second elision stage
- 2026-09-07 · e8c1a29* · mutant killed · exit 1 · `internal/mcp/tools.go` · the funnel stops bounding anything, so a refusal larger than the advertised ceiling goes out whole — the state ADR-032 shipped in, where errorResult carried no size check at all. RETRY: the first attempt of this mutant SURVIVED because the fixture s five refusals were all small enough to fit unaided · acceptance-sha256:b372ba8dfb64d297b8ccf0fd27d7fd7fc2499fab220ec6457faa367a879bc240 · covers:every tool result inside the advertised ceiling, refusals included
- 2026-09-07 · e8c1a29* · mutant survived · exit 0 · `internal/mcp/tools.go` · the terminal branch goes back to saying "nothing was written" for an applied plan. Expected to SURVIVE: the pre-apply guard makes the branch unreachable for an applied write, so no hermetic fixture reaches it. Recorded rather than dropped — the branch is deliberate belt-and-braces and this says plainly that nothing proves it · acceptance-sha256:b372ba8dfb64d297b8ccf0fd27d7fd7fc2499fab220ec6457faa367a879bc240 · covers:a write refused rather than applied when its verdict cannot be reported
  ```
  the fence passed with the mechanism broken; it may not materialize, compile, load, or assert on the changed path
  ```
- 2026-09-07 · 2ca098c* · mutant killed · exit 1 · `internal/mcp/tools.go` · the terminal branch goes back to asking Applied, so a PARTIAL application — earlier files already renamed, a later one failed — is reported as nothing written. The second HIGH of the second Codex review, and the same denial as the first HIGH one branch over · acceptance-sha256:8366a32705beb234ad459b114e5056cdcf942460e45b52c5b638ca7c41ede477 · covers:a write refused rather than applied when its verdict cannot be reported
- 2026-09-07 · 2ca098c* · mutant killed · exit 1 · `internal/mcp/tools.go` · the floor goes back to a plausible width rather than a proven one, so a plan wide enough to render more digits passes the guard and is then replaced by the funnel generic refusal, which does not say the write happened · acceptance-sha256:8366a32705beb234ad459b114e5056cdcf942460e45b52c5b638ca7c41ede477 · covers:a write refused rather than applied when its verdict cannot be reported
- 2026-09-07 · 2ca098c* · mutant killed · exit 1 · `internal/mcp/tools.go` · the pre-apply guard is gone again — re-run against the fence extended in this commit; a mutant bound to a digest nobody runs is evidence about nothing · acceptance-sha256:8366a32705beb234ad459b114e5056cdcf942460e45b52c5b638ca7c41ede477
- 2026-09-07 · 2ca098c* · mutant killed · exit 1 · `internal/mcp/tools.go` · the funnel stops bounding anything again — re-run against the fence extended in this commit; a mutant bound to a digest nobody runs is evidence about nothing · acceptance-sha256:8366a32705beb234ad459b114e5056cdcf942460e45b52c5b638ca7c41ede477
- 2026-09-07 · 2ca098c* · mutant killed · exit 1 · `internal/mcp/tools.go` · the second elision stage stops dropping file records again — re-run against the fence extended in this commit; a mutant bound to a digest nobody runs is evidence about nothing · acceptance-sha256:8366a32705beb234ad459b114e5056cdcf942460e45b52c5b638ca7c41ede477

## Invariants
- ⚠ A WRITE IS NEVER APPLIED UNDER A CEILING THAT CANNOT REPORT IT. The refusal happens before `apply.Apply`, so the tree is untouched — ADR-004's rule, applied to a budget rather than to a failure.
- The terminal branch of `boundedReceipt` is honest on its own, without depending on the pre-apply guard. A verdict whose correctness rests on a guard somewhere else is the kind that comes back.
- `withinCeiling` shrinks no answer that this CALL licensed: the served read measures and refuses before `seen.Record`, a page measures before `hold`, and `matchIndex` records nothing. ⚠ It is NOT true that such an answer licensed nothing at all — `promote(root, a.Ack)` runs at the top of both handlers and writes the PREVIOUS page into the ledger before anything is composed. So a rewritten answer can follow a licence granted earlier in the same call, for lines the caller really did receive. An earlier version of this line said "licensed nothing" flatly, which is false; the property that matters is that no pending record ever describes content the caller was not sent. Codex, second review of #135.
- A JSON-RPC error carries no `result` member and is therefore outside the advertised tool-result ceiling. That is what makes an honest answer possible at budget 0.
- `internal/read`, `internal/apply`, `internal/plan`, `internal/seen` and `internal/state` stay byte-identical against the merge base, and `go.mod` declares exactly one requirement.

## Risks

- The pre-apply guard makes a write impossible at a very small ceiling. Intended, and visible: the refusal names the minimum. A caller who sets 64 has asked for a server that cannot answer, which is their choice made legible rather than a silent failure.
- `withinCeiling` could mask a bug by shrinking an answer that should have been composed smaller. Mitigated by S5's ordering test and by leaving every existing pre-composition check in place; the funnel is a floor, not a replacement.

## Stop Condition

Stop and ask if making the funnel safe requires the served-read path to record its ledger entry
before the size check — that inverts ADR-002 and is a different decision.

## Out of Scope

- `matchIndex`'s minimum-one entry, which can still exceed the ceiling before the funnel trims it (permanent: boundary: the funnel now catches it, and shrinking the index below one entry would answer a question nobody asked; citation: file `internal/mcp/tools.go:1068`)
- A general audit of every other MCP result shape for size (deferred: `docs/adr/BACKLOG.md`)

## Verification Log
- 2026-09-07 · e8c1a29* · exit 0 · `set -o pipefail …` · acceptance-sha256:b372ba8dfb64d297b8ccf0fd27d7fd7fc2499fab220ec6457faa367a879bc240 · ms:36261
- 2026-09-07 · e8c1a29* · exit 0 · `set -o pipefail …` · acceptance-sha256:b372ba8dfb64d297b8ccf0fd27d7fd7fc2499fab220ec6457faa367a879bc240 · ms:36799
- 2026-09-07 · e8c1a29* · exit 0 · `set -o pipefail …` · acceptance-sha256:b372ba8dfb64d297b8ccf0fd27d7fd7fc2499fab220ec6457faa367a879bc240 · ms:35158
- 2026-09-07 · e8c1a29* · exit 0 · `set -o pipefail …` · acceptance-sha256:b372ba8dfb64d297b8ccf0fd27d7fd7fc2499fab220ec6457faa367a879bc240 · ms:54086
- 2026-09-07 · e8c1a29* · exit 0 · `set -o pipefail …` · acceptance-sha256:b372ba8dfb64d297b8ccf0fd27d7fd7fc2499fab220ec6457faa367a879bc240 · ms:51468
- 2026-09-07 · 2ca098c* · exit 0 · `set -o pipefail …` · acceptance-sha256:8366a32705beb234ad459b114e5056cdcf942460e45b52c5b638ca7c41ede477 · ms:35755
- 2026-09-07 · 2ca098c* · exit 0 · `set -o pipefail …` · acceptance-sha256:8366a32705beb234ad459b114e5056cdcf942460e45b52c5b638ca7c41ede477 · ms:40302
- 2026-09-07 · 2ca098c* · exit 0 · `set -o pipefail …` · acceptance-sha256:8366a32705beb234ad459b114e5056cdcf942460e45b52c5b638ca7c41ede477 · ms:39704
- 2026-09-07 · 2ca098c* · exit 0 · `set -o pipefail …` · acceptance-sha256:8366a32705beb234ad459b114e5056cdcf942460e45b52c5b638ca7c41ede477 · ms:39352
- 2026-09-07 · 2ca098c* · exit 0 · `set -o pipefail …` · acceptance-sha256:8366a32705beb234ad459b114e5056cdcf942460e45b52c5b638ca7c41ede477 · ms:38851

## Declared uncovered

- ⚠ **`boundedReceipt`'s applied-branch is UNREACHABLE and nothing proves it.** The mutant that
  reverts it to "nothing was written" SURVIVED, and the survivor line is in the Mutation Log above
  rather than deleted. That is the honest verdict: the pre-apply guard refuses before a write can
  reach a ceiling too small to report it, so no hermetic fixture gets there. The branch stays because
  a verdict whose correctness depends on a guard elsewhere staying correct is the kind that comes
  back — but "I could not make the fence fail for that reason" is the finding, not a weaker mutant.

- ⚠ **THE MUTATION LOG CARRIES TWO FENCE DIGESTS, and the earlier one is about a fence that no
  longer exists.** `b372ba8d…` is the fence before this task's follow-up commit added
  `TestAPartialApplicationIsNotReportedAsNothingWritten` and `TestTheWriteFloorIsAFloor` to it;
  `8366a327…` is the fence as it stands. Every mechanism still present was re-run against the new
  digest and killed — the pre-apply guard, the funnel, the second elision stage, the proven floor and
  the written-file test — so no live claim rests on the stale digest. The `b372ba8d` rows are left in
  place because the log is append-only and because they record something true at the time, including
  the funnel mutant that SURVIVED before its test was strengthened.
  One `b372ba8d` row is about code that is gone: the survivor for `if res.Applied {` in the terminal
  branch, which is now `if written > 0 {`. It is not evidence about anything in the tree today, and
  saying so here is cheaper than letting a future reader re-derive it.

- ⚠ **§70's UTF-8 byte count CANNOT BE MADE TO BIND from a green run, and the row says so itself.**
  A correct server enforces the ceiling in bytes, so every answer it delivers has bytes ≤ ceiling and
  therefore code points ≤ bytes ≤ ceiling: both counts pass. The unit only diverges against a server
  that has already shipped an oversized answer — the case this row exists to catch and the one no
  passing suite can stage. Reverting `.encode("utf-8")` leaves the contract green. The correction is
  kept because it is right, not because anything here proves it; the non-ASCII fixture is kept
  because it exercises multi-byte content through read, plan and write, which nothing did before.
