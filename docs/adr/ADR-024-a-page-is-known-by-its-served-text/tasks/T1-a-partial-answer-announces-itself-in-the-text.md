# Task ADR-024-T1: A partial answer announces itself in the served text and is not flagged an error

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M (multi-file)
**Owner:** Zy
**Produces:** `pagedResult()`, `indexResult()` and the served-read path return `isError` absent (T2)
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the absence of isError on an answer that served content`, `the -- PARTIAL: notice in content[0]`, `the next_read field in content[1]`, `the per-path -- <path>: <reason> lines in content[0]`, `errorResult keeping its flag`

## Goal

Stop every answer that SERVED something from claiming to be a failure, and move the promise the flag
carried onto the served text that a host does not rewrite.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/tools.go` | edit | `pagedResult` (`:637`), `indexResult` (`:814`) and the served-read return (`:319`, with its size probe at `:841`) leave `isError` absent. `errorResult` (`:422`) and the no-match-with-walk-problem branch (`:202`) are NOT touched — each delivered nothing of what was asked. This is the change. |
| `internal/mcp/tools_test.go` | edit | `TestAnOversizedReadStillReadsAsIncomplete` asserts the retired promise at line 565 and is rewritten to assert the replacement; the new `TestAPageIsKnownByItsServedText` is added here. |
| `internal/mcp/conformance_test.go` | edit | Line 579 uses `isError` as a FIXTURE GUARD — "this fixture exists to produce a page" — not as the promise. It must detect a page by `next_read` instead, or it fails for the wrong reason and hides whatever it was guarding. |
| `internal/mcp/instructions.go` | edit | Line 98 tells every host "the lines that fit, isError true, and next_read naming the spec". That sentence is what SELECTS the behaviour for a reader of the surface; leaving it makes the tool document a promise it no longer keeps. |
| `internal/mcp/schema.go` | edit | The `next_read` description ends "A paged answer is also `isError: true`". Same reason: the schema is the caller-facing declaration. |
| `internal/mcp/mcp.go` | edit | `:260`, the `tools/list` description a host reads BEFORE it calls anything, still promised "isError true" for a page. Missed on the first pass and found by the Codex review of #118, because neither §48 nor the first §62 inspected it. |
| `scripts/contract.sh` | edit | Three EXISTING rows pin the retired promise against the built binary — `:2279` (the index), `:3137` (taught-versus-shipped) and `:3367`/`:3368` (ADR-023's shapes). They are moved onto the new promise here rather than in T2, because they went red the moment the behaviour changed and a red contract cannot be carried between tasks. T2 still owns the NEW row, §62. |

## Ordered Steps

1. [S1] Write `TestAPageIsKnownByItsServedText` in `internal/mcp/tools_test.go` and confirm it is RED against the current code. It asserts all four members of the class in one place: a paged read has no `isError` and its `content[0]` carries `-- PARTIAL:` with the served range and remaining count while `content[1]` carries `next_read`; an oversized grep index has no `isError`; a read that served content alongside an unreadable path has no `isError` and names that path in `content[0]`; and a real refusal (`exclude` without `grep`, which serves nothing) is still `isError: true`, so the two kinds cannot collapse into one.
2. [S2] Change `pagedResult` in `internal/mcp/tools.go` so `isError` is absent, keeping the report and receipt as they are. [proof: mutation]
3. [S3] Change `indexResult` in `internal/mcp/tools.go` the same way, leaving `errorResult` alone. [proof: mutation]
4. [S10] Change the served-read return at `internal/mcp/tools.go:319` to pass `false` rather than `problems > 0`, so a read that served content alongside an unusable path is not flagged, and mirror it in `servedOrIndex`'s size probe at `:841` so the probe measures the shape that is actually sent. Leave the `:202` branch alone — it matched nothing and served no content, and its own comment already draws that line. [proof: mutation]
5. [S4] Rewrite `TestAnOversizedReadStillReadsAsIncomplete` so it asserts the replacement promise — the served text says a part was served and names the continuation — instead of the flag. Do not delete it: the behaviour it guards still needs a test.
6. [S5] Repair the fixture guard at `internal/mcp/conformance_test.go:579` to detect a page by the presence of `next_read` in its receipt rather than by `isError`.
7. [S6] Update `internal/mcp/instructions.go:98` and the `next_read` description in `internal/mcp/schema.go` so the surface no longer promises a flag it does not set. [proof: acceptance]
8. [S7] Rewrite `TestAnOversizedGrepReturnsTheIndexAndNotADeadEnd` (`:663`) and `TestAWalkProblemSurvivesAValidSibling` (`:898`) to assert the served text rather than the flag. The first is ADR-017's Enforced-by test; it keeps enforcing ADR-017 — that an oversized grep answers with a usable index — and stops asserting the clause ADR-024 invalidates. The second serves its good sibling, so it is a served answer.
9. [S8] Leave four assertions untouched, because each names an answer that SERVED NOTHING and therefore keeps its flag: `TestAReadOverTheLimitIsRefusedNotTruncated` (`:305`), `TestGrepRefusesARangedSpec` (`:641`), `TestAfterWithoutGrepIsRefused` (`:978`), and `TestAWalkProblemIsReportedAndNotSwallowed` (`:861`) — the last of these walks only an unusable path and matches nothing, taking the `:202` branch whose own comment already draws this line: "a walk that could not LOOK somewhere is a different answer again, and it is an error". Confirm by running them. [proof: acceptance]
10. [S9] Run the full package and the engine go/no-go. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/mcp/ -run 'TestAPageIsKnownByItsServedText' -count=1 2>&1 | tee /tmp/adr024-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr024-t1.out \
  && go test ./internal/mcp/ -count=1 -run 'TestAnOversizedReadStillReadsAsIncomplete|TestAnOversizedGrepReturnsTheIndexAndNotADeadEnd|TestAWalkProblemIsReportedAndNotSwallowed|TestAWalkProblemSurvivesAValidSibling|TestAReadOverTheLimitIsRefusedNotTruncated|TestAPagedReadReassemblesTheWholeFile|TestAReadResultCarriesNoStructuredContent' \
  && go test ./internal/mcp/... -count=1 \
  && gofmt -l internal/mcp \
  && go vet ./internal/mcp/...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPageIsKnownByItsServedText` | `internal/mcp/tools_test.go` | All four members in one place: page, index and served-with-problems carry no `isError` and say so in `content[0]`; a refusal that served nothing still carries it | — | S1, S2, S3 |
| `TestAnOversizedReadStillReadsAsIncomplete` | `internal/mcp/tools_test.go` | A caller that stops at a page can tell it does not have the file, from the served text alone | — | S4 |
| `TestAnOversizedGrepReturnsTheIndexAndNotADeadEnd` | `internal/mcp/tools_test.go` | ADR-017 still holds — an oversized grep answers with a usable index — without asserting the retired flag | — | S7 |
| `TestAWalkProblemIsReportedAndNotSwallowed` | `internal/mcp/tools_test.go` | Unchanged: a walk that served nothing and could not look where it was told is still `isError: true` | — | S8 |
| `TestAWalkProblemSurvivesAValidSibling` | `internal/mcp/tools_test.go` | A good sibling is served and the bad path is still counted and named | — | S7 |
| `TestAReadOverTheLimitIsRefusedNotTruncated` | `internal/mcp/tools_test.go` | Unchanged: a refusal that served nothing is still `isError: true` | — | S8 |
| `TestAReadResultCarriesNoStructuredContent` | `internal/mcp/conformance_test.go` | ADR-023 still holds, and its page fixture still reaches the paging path after the guard changes | — | S5 |
| `TestAPagedReadReassemblesTheWholeFile` | `internal/mcp/tools_test.go` | Unchanged: following `next_read` still yields the whole file byte-for-byte | — | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestAPageIsKnownByItsServedText` |
| 2 — something selects it | `firstPage` is the only caller of `pagedResult`, `servedOrIndex` the only caller of `indexResult`, and `:319` the ordinary served-read return; the mutations at S2/S3 restore `IsError: true` and the fence must go red |
| 3 — the caller can discover it | `internal/mcp/schema.go`'s `next_read` description and `internal/mcp/instructions.go` are the declared interface, updated in S6; `TestEveryDeclaredOutputSchemaValidatesARealResponse` reads the schema |
| 4 — it is used | Observed on the served path: the A/B in the ADR's Context is the measurement, and contract §62 (T2) re-runs its shape against the built binary on every contract run |

## Mutation Log

- 2026-09-06 · 54d6b18* · mutant killed · exit 1 · `internal/mcp/tools.go` · restores the flag on a page; the fence must go red because a host then discards the middle and the test asserts the flag is absent · acceptance-sha256:b4defa76d887e81ca41340d38771ee1a0d51c70fa007f27a7547505c8fedc9e4 · covers:the absence of isError on an answer that served content
- 2026-09-06 · 54d6b18* · mutant killed · exit 1 · `internal/mcp/tools.go` · restores the flag on an oversized grep index; the fence must go red because the index is a served answer and the test asserts the flag is absent · acceptance-sha256:b4defa76d887e81ca41340d38771ee1a0d51c70fa007f27a7547505c8fedc9e4 · covers:the absence of isError on an answer that served content
- 2026-09-06 · 54d6b18* · mutant killed · exit 1 · `internal/mcp/tools.go` · restores the flag on an ordinary read that served content beside an unusable path — the most exposed member, since it needs no oversized file; the fence must go red · acceptance-sha256:b4defa76d887e81ca41340d38771ee1a0d51c70fa007f27a7547505c8fedc9e4 · covers:the absence of isError on an answer that served content
- 2026-09-06 · 54d6b18* · mutant killed · exit 1 · `internal/mcp/tools.go` · lowercases the PARTIAL notice a page carries in content[0]; the fence must go red because the notice IS the replacement promise now that the flag is gone · acceptance-sha256:b4defa76d887e81ca41340d38771ee1a0d51c70fa007f27a7547505c8fedc9e4 · covers:the -- PARTIAL: notice in content[0]
- 2026-09-06 · 54d6b18* · mutant killed · exit 1 · `internal/mcp/tools.go` · empties the continuation spec a page names in content[1]; the fence must go red because a page a caller cannot continue is the dead end ADR-014 removed · acceptance-sha256:b4defa76d887e81ca41340d38771ee1a0d51c70fa007f27a7547505c8fedc9e4 · covers:the next_read field in content[1]
- 2026-09-06 · 54d6b18* · mutant killed · exit 1 · `internal/mcp/tools.go` · drops the flag from a genuine refusal; the fence must go red because ADR-024 removes it only from answers that SERVED something, and without this the change would collapse both shapes into one · acceptance-sha256:b4defa76d887e81ca41340d38771ee1a0d51c70fa007f27a7547505c8fedc9e4 · covers:errorResult keeping its flag
- 2026-09-06 · 54d6b18* · mutant killed · exit 1 · `internal/mcp/tools.go` · stops a served answer naming the path it could not use; the fence must go red because dropping the flag must not also drop the report, which is now the only place a caller learns a path failed · acceptance-sha256:b4defa76d887e81ca41340d38771ee1a0d51c70fa007f27a7547505c8fedc9e4 · covers:the per-path -- <path>: <reason> lines in content[0]
- 2026-09-06 · 54d6b18* · mutant killed · exit 1 · `internal/mcp/tools.go` · restores the flag on a page; the fence must go red · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · covers:the absence of isError on an answer that served content
- 2026-09-06 · 54d6b18* · mutant killed · exit 1 · `internal/mcp/tools.go` · restores the flag on an oversized grep index; the fence must go red · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · covers:the absence of isError on an answer that served content
- 2026-09-06 · 54d6b18* · mutant killed · exit 1 · `internal/mcp/tools.go` · restores the flag on an ordinary read that served content beside an unusable path — the most exposed member; the fence must go red · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · covers:the absence of isError on an answer that served content
- 2026-09-06 · 54d6b18* · mutant killed · exit 1 · `internal/mcp/tools.go` · lowercases the PARTIAL notice a page carries in content[0]; that notice IS the replacement promise now the flag is gone · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · covers:the -- PARTIAL: notice in content[0]
- 2026-09-06 · 54d6b18* · mutant killed · exit 1 · `internal/mcp/tools.go` · empties the continuation spec; a page a caller cannot continue is the dead end ADR-014 removed · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · covers:the next_read field in content[1]
- 2026-09-06 · 54d6b18* · mutant killed · exit 1 · `internal/mcp/tools.go` · drops the flag from a genuine refusal; ADR-024 removes it only from answers that SERVED something · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · covers:errorResult keeping its flag
- 2026-09-06 · 54d6b18* · mutant killed · exit 1 · `internal/mcp/tools.go` · stops a served answer naming the path it could not use; dropping the flag must not drop the report · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · covers:the per-path -- <path>: <reason> lines in content[0]
- 2026-09-06 · 54a7e59* · mutant killed · exit 1 · `internal/mcp/tools.go` · restores the flag on a page; the fence must go red · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · covers:the absence of isError on an answer that served content
- 2026-09-06 · 54a7e59* · mutant killed · exit 1 · `internal/mcp/tools.go` · restores the flag on an oversized grep index; the fence must go red · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · covers:the absence of isError on an answer that served content
- 2026-09-06 · 54a7e59* · mutant killed · exit 1 · `internal/mcp/tools.go` · restores the flag on an ordinary read that served content beside an unusable path; the fence must go red · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · covers:the absence of isError on an answer that served content
- 2026-09-06 · 54a7e59* · mutant killed · exit 1 · `internal/mcp/tools.go` · lowercases the notice a page carries in content[0], now asserted against content[0] alone · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · covers:the -- PARTIAL: notice in content[0]
- 2026-09-06 · 54a7e59* · mutant killed · exit 1 · `internal/mcp/tools.go` · empties the continuation spec; a page a caller cannot continue is the dead end ADR-014 removed · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · covers:the next_read field in content[1]
- 2026-09-06 · 54a7e59* · mutant killed · exit 1 · `internal/mcp/tools.go` · drops the flag from a genuine refusal; ADR-024 removes it only from answers that delivered what was asked · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · covers:errorResult keeping its flag
- 2026-09-06 · 54a7e59* · mutant killed · exit 1 · `internal/mcp/tools.go` · stops a served answer naming the path it could not use; dropping the flag must not drop the report · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · covers:the per-path -- <path>: <reason> lines in content[0]
- 2026-09-06 · 6c4326d* · mutant killed · exit 1 · `internal/mcp/tools.go` · restores the flag on a page; the fence must go red · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · covers:the absence of isError on an answer that served content
- 2026-09-06 · 6c4326d* · mutant killed · exit 1 · `internal/mcp/tools.go` · restores the flag on an oversized grep index; the fence must go red · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · covers:the absence of isError on an answer that served content
- 2026-09-06 · 6c4326d* · mutant killed · exit 1 · `internal/mcp/tools.go` · restores the flag on an ordinary read that served content beside an unusable path; the fence must go red · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · covers:the absence of isError on an answer that served content
- 2026-09-06 · 6c4326d* · mutant killed · exit 1 · `internal/mcp/tools.go` · lowercases the notice a page carries in content[0] · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · covers:the -- PARTIAL: notice in content[0]
- 2026-09-06 · 6c4326d* · mutant killed · exit 1 · `internal/mcp/tools.go` · empties the continuation spec a page names in content[1] · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · covers:the next_read field in content[1]
- 2026-09-06 · 6c4326d* · mutant killed · exit 1 · `internal/mcp/tools.go` · drops the flag from a genuine refusal · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · covers:errorResult keeping its flag
- 2026-09-06 · 6c4326d* · mutant killed · exit 1 · `internal/mcp/tools.go` · stops a served answer naming the path it could not use · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · covers:the per-path -- <path>: <reason> lines in content[0]

## Invariants

- `errorResult` still sets `isError: true`. A refusal is an error and must stay one.
- A page still carries `next_read` in `content[1]`, and the whole file is still reassemblable by following it — `TestAPagedReadReassemblesTheWholeFile` must stay green untouched.
- The ledger still records only the span the page served (ADR-014 Decision 3, ADR-002). This task changes no ledger code.
- The engine packages stay byte-identical against the merge base: `internal/read`, `internal/apply`, `internal/plan`, `internal/seen`, `internal/check`, `internal/state`.
- `go.mod` still declares exactly one requirement.

## Risks

- A host that used `isError` as its only cue shows partial answers less prominently. Mitigated: the `-- PARTIAL:` line is in the rendered text and the test asserts its content, not merely its presence.
- The grep-index half is unmeasured on a real host. Mitigated: it shares the flag, the code path and the host behaviour with the page; the ADR says plainly it was not separately measured.

## Stop Condition

Stop and ask if `TestAPagedReadReassemblesTheWholeFile` goes red — that would mean the change reached
the continuation mechanism rather than only the flag, which is out of scope here. Stop also if a
paged read still arrives gapped on a host with `isError` absent: that falsifies the ADR's decision,
and the record must be revised rather than the test relaxed.

## Out of Scope

- The ledger recording what was SENT rather than SEEN (deferred: `docs/adr/BACKLOG.md` under ADR-023)
- Making `MaxResultChars` caller-set (deferred: `docs/adr/BACKLOG.md` under ADR-023)
- The contract row — that is T2's job

## Verification Log
- 2026-09-06 · 54d6b18* · exit 1 · `set -o pipefail …` · acceptance-sha256:b4defa76d887e81ca41340d38771ee1a0d51c70fa007f27a7547505c8fedc9e4 · ms:1119
  ```
  --- last 7 line(s) of stdout
  --- FAIL: TestAPageIsKnownByItsServedText (0.04s)
      tools_test.go:999: a paged read is marked isError; a host reads that as a failed call and discards the middle
      tools_test.go:1015: an oversized grep index is marked isError; it served an index and is exposed to the same truncation
      tools_test.go:1036: a read that served its good sibling is marked isError; the content it did serve is what a host then truncates
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.394s
  FAIL
  ```
- 2026-09-06 · 54d6b18* · exit 0 · `set -o pipefail …` · acceptance-sha256:b4defa76d887e81ca41340d38771ee1a0d51c70fa007f27a7547505c8fedc9e4 · ms:6258
- 2026-09-06 · 54d6b18* · exit 0 · `set -o pipefail …` · acceptance-sha256:b4defa76d887e81ca41340d38771ee1a0d51c70fa007f27a7547505c8fedc9e4 · ms:5641
- 2026-09-06 · 54d6b18* · exit 0 · `set -o pipefail …` · acceptance-sha256:b4defa76d887e81ca41340d38771ee1a0d51c70fa007f27a7547505c8fedc9e4 · ms:6691
- 2026-09-06 · 54d6b18* · exit 0 · `set -o pipefail …` · acceptance-sha256:b4defa76d887e81ca41340d38771ee1a0d51c70fa007f27a7547505c8fedc9e4 · ms:9940
- 2026-09-06 · 54d6b18* · exit 0 · `set -o pipefail …` · acceptance-sha256:b4defa76d887e81ca41340d38771ee1a0d51c70fa007f27a7547505c8fedc9e4 · ms:7215
- 2026-09-06 · 54d6b18* · exit 0 · `set -o pipefail …` · acceptance-sha256:b4defa76d887e81ca41340d38771ee1a0d51c70fa007f27a7547505c8fedc9e4 · ms:5719
- 2026-09-06 · 54d6b18* · exit 0 · `set -o pipefail …` · acceptance-sha256:b4defa76d887e81ca41340d38771ee1a0d51c70fa007f27a7547505c8fedc9e4 · ms:5282
- 2026-09-06 · 54d6b18* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · ms:6438
- 2026-09-06 · 54d6b18* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · ms:5646
- 2026-09-06 · 54d6b18* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · ms:5614
- 2026-09-06 · 54d6b18* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · ms:7818
- 2026-09-06 · 54d6b18* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · ms:8178
- 2026-09-06 · 54d6b18* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · ms:6875
- 2026-09-06 · 54d6b18* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · ms:6515
- 2026-09-06 · 54a7e59* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · ms:6695
- 2026-09-06 · 54a7e59* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · ms:9798
- 2026-09-06 · 54a7e59* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · ms:6378
- 2026-09-06 · 54a7e59* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · ms:5854
- 2026-09-06 · 54a7e59* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · ms:5825
- 2026-09-06 · 54a7e59* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · ms:5975
- 2026-09-06 · 54a7e59* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · ms:5902
- 2026-09-06 · 6c4326d* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · ms:6735
- 2026-09-06 · 6c4326d* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · ms:6015
- 2026-09-06 · 6c4326d* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · ms:5642
- 2026-09-06 · 6c4326d* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · ms:5895
- 2026-09-06 · 6c4326d* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · ms:5796
- 2026-09-06 · 6c4326d* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · ms:8179
- 2026-09-06 · 6c4326d* · exit 0 · `set -o pipefail …` · acceptance-sha256:9c27e92dcee4c9a7c7def518fd09a2dbab8078ef35abe061165f2be82fcdd34b · ms:8477
