# Task ADR-025-T1: The flag returns when a read served nothing

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S (single file, plus its test)
**Owner:** Zy
**Produces:** the served-read return flags when `len(observed) == 0` (T2)
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the flag on a read that served nothing`, `the absence of the flag on a read that served something`, `the per-path -- <path>: <reason> lines surviving on the flagged result`

## Goal

Make an `mrw_read` that delivered none of what was asked for report itself as an error, so the
served-read return agrees with the walk branch that already does.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/tools.go` | edit | The served-read return at `:320` passes `len(observed) == 0` instead of `false`. This is the whole change. `errorResult`, `pagedResult`, `indexResult` and the `:202` branch are NOT touched. |
| `internal/mcp/tools_test.go` | edit | `TestAReadThatServedNothingIsAnError` is added here. It is what SELECTS the new condition — the only test that reaches the served-read return with an empty `observed`. |

## Ordered Steps

1. [S1] Write `TestAReadThatServedNothingIsAnError` in `internal/mcp/tools_test.go` and confirm it is RED against the current code. Four shapes in one test, so that no server can satisfy it by flagging everything or nothing: a read naming only an unusable path is `isError: true` and still names that path with its reason in `content[0]`; a read naming an unusable path ALONGSIDE a good one served the good one and is NOT flagged (ADR-024's member, which must not regress); a range that matches nothing on a real file is NOT flagged, because the file was observed even though no line was served; and an empty file addressed by a range is NOT flagged, for the same reason.
2. [S2] Change the served-read return at `internal/mcp/tools.go:320` from `false` to `len(observed) == 0`, and say in the comment that the observation count is the whole test — a spec that served no lines is still observed, so this cannot catch an empty file or a missed range. [proof: mutation]
3. [S3] Confirm ADR-024's own test is still green untouched: `TestAPageIsKnownByItsServedText`'s four members all served something, so none of them may change. [proof: acceptance]
4. [S4] Confirm the walk branch is untouched by running `TestAWalkProblemIsReportedAndNotSwallowed` and `TestAWalkProblemSurvivesAValidSibling` — the first takes `:202` and keeps its flag, the second serves a sibling and keeps none. [proof: acceptance]
5. [S5] Run the whole package, gofmt and vet, and confirm the engine packages are byte-identical against the merge base. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/mcp/ -run 'TestAReadThatServedNothingIsAnError' -count=1 -v 2>&1 | tee /tmp/adr025-t1.out \
  && grep -q '^--- PASS: TestAReadThatServedNothingIsAnError' /tmp/adr025-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr025-t1.out \
  && go test ./internal/mcp/ -count=1 -run 'TestAPageIsKnownByItsServedText|TestAWalkProblemIsReportedAndNotSwallowed|TestAWalkProblemSurvivesAValidSibling|TestAPagedReadReassemblesTheWholeFile' \
  && go test ./internal/mcp/... ./internal/adversarial/ -count=1 \
  && [ -z "$(gofmt -l internal/mcp)" ] \
  && go vet ./internal/mcp/...
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAReadThatServedNothingIsAnError` | `internal/mcp/tools_test.go` | A read that delivered nothing is flagged and still names the path and reason; one that served a sibling, one whose range missed, and one against an empty file are all unflagged | — | S1, S2 |
| `TestAPageIsKnownByItsServedText` | `internal/mcp/tools_test.go` | Unchanged: ADR-024's four served members stay unflagged and a refusal stays flagged | — | S3 |
| `TestAWalkProblemIsReportedAndNotSwallowed` | `internal/mcp/tools_test.go` | Unchanged: the `:202` branch still flags | — | S4 |
| `TestAWalkProblemSurvivesAValidSibling` | `internal/mcp/tools_test.go` | Unchanged: a served sibling is still unflagged and the bad path still counted | — | S4 |
| `TestAPagedReadReassemblesTheWholeFile` | `internal/mcp/tools_test.go` | Unchanged: paging is untouched by this record | — | — |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `TestAReadThatServedNothingIsAnError` |
| 2 — something selects it | The served-read return at `:320` is the ordinary read path — every `mrw_read` that is not paged, indexed or refused returns through it; the S2 mutation restores `false` and the fence must go red |
| 3 — the caller can discover it | `isError` is the MCP protocol's own field, declared by the protocol rather than by this server's schema; the flagged result's `content[0]` still carries the per-path reasons, asserted in S1 |
| 4 — it is used | Contract §63 (T2) re-runs the shape against the built binary on every contract run; no telemetry, per ADR-009 |

## Mutation Log

- 2026-09-06 · d8a1d0f* · mutant killed · exit 1 · `internal/mcp/tools.go` · restores the unconditional false ADR-024 left behind, so a read that served nothing comes back unflagged again; the fence must go red or the new condition is asserted by nothing · acceptance-sha256:9a92b75a345aaa0499bb362ae0d6a6570ebebbfff5d867d616d7f44ee5f3e757 · covers:the flag on a read that served nothing
- 2026-09-06 · d8a1d0f* · mutant killed · exit 1 · `internal/mcp/tools.go` · flags every served read, which is what a server that had simply started flagging everything would do; the fence must go red on the sibling, missed-range and empty-file shapes or the test could be satisfied without discriminating · acceptance-sha256:9a92b75a345aaa0499bb362ae0d6a6570ebebbfff5d867d616d7f44ee5f3e757 · covers:the absence of the flag on a read that served something
- 2026-09-06 · d8a1d0f* · mutant killed · exit 1 · `internal/mcp/tools.go` · empties the served report, so the flagged answer no longer names the path it could not use; the fence must go red because flagging a result must not be traded for dropping the only thing that result carries · acceptance-sha256:9a92b75a345aaa0499bb362ae0d6a6570ebebbfff5d867d616d7f44ee5f3e757 · covers:the per-path -- <path>: <reason> lines surviving on the flagged result
- 2026-09-06 · d8a1d0f* · mutant killed · exit 1 · `internal/mcp/tools.go` · restores the unconditional false ADR-024 left behind, so a read that served nothing comes back unflagged; the fence must go red or the new condition is asserted by nothing · acceptance-sha256:fef6f4f3f7c03f715e468566ee1c1e6a8d51bc452ab0af20ee3f90cf2e144c66 · covers:the flag on a read that served nothing
- 2026-09-06 · d8a1d0f* · mutant killed · exit 1 · `internal/mcp/tools.go` · flags every served read, which is what a server that had simply started flagging everything would do; the fence must go red on the sibling, missed-range and empty-file shapes · acceptance-sha256:fef6f4f3f7c03f715e468566ee1c1e6a8d51bc452ab0af20ee3f90cf2e144c66 · covers:the absence of the flag on a read that served something
- 2026-09-06 · d8a1d0f* · mutant killed · exit 1 · `internal/mcp/tools.go` · empties the served report, so the flagged answer no longer names the path it could not use; flagging must not be traded for dropping the only thing the answer carries · acceptance-sha256:fef6f4f3f7c03f715e468566ee1c1e6a8d51bc452ab0af20ee3f90cf2e144c66 · covers:the per-path -- <path>: <reason> lines surviving on the flagged result

## Invariants

- `errorResult` still sets `isError: true`.
- ADR-024's four served members stay unflagged: a page, an oversized grep index, a read that served content beside an unusable path, and the size probe's shape.
- The `:202` no-match branch is untouched and still flags on `len(walkProblems) > 0`.
- `problems` and the report text in `content[0]` are unchanged on every shape, flagged or not.
- The engine packages stay byte-identical against the merge base: `internal/read`, `internal/apply`, `internal/plan`, `internal/seen`, `internal/check`, `internal/state`.
- `go.mod` still declares exactly one requirement.

## Risks

- The condition could be widened to `problems > 0` by a later editor, which silently restores what ADR-024 removed. Mitigated: `TestAPageIsKnownByItsServedText`'s served-with-problems member goes red if it happens, and this task's own test adds the missed-range and empty-file shapes, which that widening would also flag.
- `internal/adversarial/record_test.go` turns `go test ./...` red the moment a Tests table names a test that does not exist, so this task's table is red until S1 lands. That is the red-first property working, and the fence runs `./internal/adversarial/` so the task's own gate can see it rather than leaving it to the repository-wide run.

## Stop Condition

Stop and ask if `TestAPageIsKnownByItsServedText` goes red — that means the change reached a shape
that served something, which is ADR-024's territory and not this record's. Stop also if a served
answer beside an unusable path starts arriving truncated on a host once flagged, which would falsify
the Decision's falsifiability paragraph rather than the test.

## Out of Scope

- The contract row — that is T2's job
- The `:202` branch's flag (it already behaves this way)

## Verification Log
- 2026-09-06 · d8a1d0f* · exit 1 · `set -o pipefail …` · acceptance-sha256:9a92b75a345aaa0499bb362ae0d6a6570ebebbfff5d867d616d7f44ee5f3e757 · ms:920
  ```
  --- last 6 line(s) of stdout
  === RUN   TestAReadThatServedNothingIsAnError
      tools_test.go:1104: a read that served nothing is not marked isError; the caller got none of what it asked for and cannot tell that from the envelope
  --- FAIL: TestAReadThatServedNothingIsAnError (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.157s
  FAIL
  ```
- 2026-09-06 · d8a1d0f* · exit 0 · `set -o pipefail …` · acceptance-sha256:9a92b75a345aaa0499bb362ae0d6a6570ebebbfff5d867d616d7f44ee5f3e757 · ms:5509
- 2026-09-06 · d8a1d0f* · exit 0 · `set -o pipefail …` · acceptance-sha256:9a92b75a345aaa0499bb362ae0d6a6570ebebbfff5d867d616d7f44ee5f3e757 · ms:4136
- 2026-09-06 · d8a1d0f* · exit 0 · `set -o pipefail …` · acceptance-sha256:9a92b75a345aaa0499bb362ae0d6a6570ebebbfff5d867d616d7f44ee5f3e757 · ms:3747
- 2026-09-06 · d8a1d0f* · exit 0 · `set -o pipefail …` · acceptance-sha256:9a92b75a345aaa0499bb362ae0d6a6570ebebbfff5d867d616d7f44ee5f3e757 · ms:4761
- 2026-09-06 · d8a1d0f* · exit 0 · `set -o pipefail …` · acceptance-sha256:fef6f4f3f7c03f715e468566ee1c1e6a8d51bc452ab0af20ee3f90cf2e144c66 · ms:6434
- 2026-09-06 · d8a1d0f* · exit 0 · `set -o pipefail …` · acceptance-sha256:fef6f4f3f7c03f715e468566ee1c1e6a8d51bc452ab0af20ee3f90cf2e144c66 · ms:5680
- 2026-09-06 · d8a1d0f* · exit 0 · `set -o pipefail …` · acceptance-sha256:fef6f4f3f7c03f715e468566ee1c1e6a8d51bc452ab0af20ee3f90cf2e144c66 · ms:5664
- 2026-09-06 · d8a1d0f* · exit 0 · `set -o pipefail …` · acceptance-sha256:fef6f4f3f7c03f715e468566ee1c1e6a8d51bc452ab0af20ee3f90cf2e144c66 · ms:5448
