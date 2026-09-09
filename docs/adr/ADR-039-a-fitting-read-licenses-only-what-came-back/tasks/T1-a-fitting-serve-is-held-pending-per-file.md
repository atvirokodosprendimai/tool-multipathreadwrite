# Task ADR-039-T1: A fitting serve is held pending, per file

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** unassigned
**Produces:** fitting serves held pending per file
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a fitting serve recording nothing until ack`, `hold naming each file's own span`, `paged ack still required`

## Goal

Stop `readTool` recording a fitting MCP serve, and hold each file's checkpointed spans pending
against that file — not against whichever path `hold` currently picks from the map.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/tools.go` | edit | The fitting return currently `seen.Record`s then returns. After `encodedSize` fits, split the report on `==> path` headers, `interleave` per file, recompose, re-measure, `hold`, do not `Record`. This line is what *selects* the new licensing. |
| `internal/mcp/ack.go` | edit | `hold` keys every checkpoint to the file whose lines it bracketed. Delete the "paged read serves one file" one-path loop. |
| `internal/mcp/ack_test.go` | edit | `TestAFittingReadLicensesOnlyWhatCameBack` and `TestAFittingReadHoldsPendingPerFile`. |
| `internal/mcp/tools_test.go` | edit | `TestAReadUnderTheLimitIsUnchanged` currently wants a ledger entry on an ordinary MCP read — that *is* the hole. Rewrite it to want checkpoints and no ledger until ack. |

## Ordered Steps

1. [S1] Write `TestAFittingReadLicensesOnlyWhatCameBack` and `TestAFittingReadHoldsPendingPerFile`, and rewrite `TestAReadUnderTheLimitIsUnchanged` so an ordinary three-line MCP read leaves no ledger entry and the served text carries `-- ck` brackets. Confirm the three are RED against current `readTool`/`hold`. ⚠ The two-file test acks only A's checkpoints and writes B; a single-file fixture is green against today's one-path `hold`. [proof: mutation]
2. [S2] Reshape `hold` so each checkpoint is stored with the path and sha of the file it brackets. Split a multi-file report on `==> path` headers before `interleave`. [proof: mutation]
3. [S3] On the fitting return in `readTool`, after the encode that will be sent: mark, re-measure `encodedSize`, overflow through the existing page/index/refusal path without holding, otherwise `hold` and do not `seen.Record`. [proof: mutation]
4. [S4] Confirm existing paged tests still pass, including `TestOnlyAckedSegmentsAreRecorded` and `TestAckOnAReadPromotesToo`. [proof: acceptance]
5. [S5] Run `gofmt` and `go vet` unpiped on `./internal/mcp`. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/mcp/ -count=1 -v \
  -run 'TestAFittingReadLicensesOnlyWhatCameBack|TestAFittingReadHoldsPendingPerFile|TestAReadUnderTheLimitIsUnchanged' 2>&1 \
  | tee /tmp/adr039-t1.out \
  && grep -q '^--- PASS: TestAFittingReadLicensesOnlyWhatCameBack' /tmp/adr039-t1.out \
  && grep -q '^--- PASS: TestAFittingReadHoldsPendingPerFile' /tmp/adr039-t1.out \
  && grep -q '^--- PASS: TestAReadUnderTheLimitIsUnchanged' /tmp/adr039-t1.out \
  && ! grep -qE "no tests to run|^FAIL|^--- FAIL" /tmp/adr039-t1.out \
  && go test ./internal/mcp/ -count=1 \
    -run 'TestOnlyAckedSegmentsAreRecorded|TestAckOnAReadPromotesToo' \
  && [ -z "$(gofmt -l internal/mcp)" ] \
  && go vet ./internal/mcp/
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAFittingReadLicensesOnlyWhatCameBack` | `internal/mcp/ack_test.go` | A three-line MCP read carries checkpoints, records nothing, a write without ack is refused, the same write with ack applies | — | S1, S3 |
| `TestAFittingReadHoldsPendingPerFile` | `internal/mcp/ack_test.go` | Two fitting files; ack only A's ids; write to B refused; write to A applies | — | S1, S2 |
| `TestAReadUnderTheLimitIsUnchanged` | `internal/mcp/tools_test.go` | An ordinary under-limit MCP read is not `isError` and no longer leaves a ledger entry until ack | — | S1, S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the three tests in the Tests table |
| 2 — something selects it | `readTool`'s fitting return is the only MCP path that still `seen.Record`s a serve. Deleting the `hold` call and restoring `Record` fails S1 |
| 3 — the caller can discover it | T3 retargets the `ack` schema; this task does not change help text |
| 4 — it is used | nothing measures this yet — ADR-009 refuses transmission of caller outcomes; §77 (T3) is the evidence the binary does it |

## Mutation Log

- 2026-09-09 · ebbc4be* · mutant killed · exit 1 · `internal/mcp/tools.go` · restoring Record-on-serve licenses a fitting MCP read before ack · acceptance-sha256:1c87dfdf9e6cce59055714f4a847a27bc4c52d19ab77a41169b74099c740963f
- 2026-09-09 · ebbc4be* · mutant killed · exit 1 · `internal/mcp/tools.go` · holding every file's spans against b.txt licenses B from A's ack · acceptance-sha256:1c87dfdf9e6cce59055714f4a847a27bc4c52d19ab77a41169b74099c740963f

## Invariants

- Paged reads still hold pending and still require ack. T1 must not "fix" pages by recording them on serve.
- `AckRule`, `ckEvery`, checkpoint randomness, and consume-on-promote are unchanged.
- `internal/read`, `internal/apply`, `internal/plan`, `internal/seen`, `internal/check`, `internal/state` stay byte-identical against the merge-base.
- Overflow after markers holds nothing.

## Risks

- Rewriting `TestAReadUnderTheLimitIsUnchanged` without a new name hides that the old assertion *was* the hole. Mitigation: the new tests carry the new names; the rewrite is called out in the parent record.
- Splitting on `==> path` must not treat `==> x  UNREADABLE` as a file to hold. Mitigation: only slices that contain numbered `NNN|` lines produce spans; `interleave` already skips non-content lines.

## Stop Condition

Stop if the only way to hold two files is a new pending format, a second marker design, or a change in `internal/seen`. Stop if `encodedSize` after markers is measured against a probe that is not the object sent (ADR-031's three-attempt failure). Stop if making this work requires locking the target file or touching `internal/apply`.

## Out of Scope

- CLI licensing (T2)
- Contract row and handshake retarget (T3)
- Grep indexes (parent Out of Scope)

## Verification Log
- 2026-09-09 · ebbc4be* · exit 0 · `set -o pipefail …` · acceptance-sha256:1c87dfdf9e6cce59055714f4a847a27bc4c52d19ab77a41169b74099c740963f · ms:1447
- 2026-09-09 · ebbc4be* · exit 0 · `set -o pipefail …` · acceptance-sha256:1c87dfdf9e6cce59055714f4a847a27bc4c52d19ab77a41169b74099c740963f · ms:832
- 2026-09-09 · ebbc4be* · exit 0 · `set -o pipefail …` · acceptance-sha256:1c87dfdf9e6cce59055714f4a847a27bc4c52d19ab77a41169b74099c740963f · ms:759
- 2026-09-09 · ebbc4be* · exit 0 · `set -o pipefail …` · acceptance-sha256:1c87dfdf9e6cce59055714f4a847a27bc4c52d19ab77a41169b74099c740963f · ms:803
