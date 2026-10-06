# Task ADR-127-T1: every landed write is counted, and a check that saw others says how many

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `writer.Writes`, `writer.bumpWrites`, `Verified.DriftWriters`, the `drift_writers` receipt key
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `every landed write is counted, and a check that saw others says how many`

## Goal

Each landed write bumps the checkout's write counter under the lock; Verify reports how many other writes landed while the check ran, on both receipts and in both texts.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/writer/writes.go` | add | the counter |
| `internal/writer/writer.go` | edit | bump under the lock, return it to Land |
| `internal/writer/flow.go` | edit | `Landed.gen`, `Verified.DriftWriters` |
| `internal/writer/writes127_test.go` | add | the tests |
| `cmd/mrw/main.go`, `internal/mcp/tools.go`, `internal/mcp/schema.go` | edit | the key and the line |
| `docs/receipts.txt` | edit | two keys |
| `scripts/contract.sh` | edit | §225 |
| `AGENTS.md` | edit | the drift paragraph |

## Ordered Steps

1. [S1] Write `TestEveryLandedWriteBumpsTheCounter` and `TestAWriteThatLandsDuringTheCheckIsCounted`. Confirm RED. [proof: mutation]
2. [S2] The counter, the bump, the reading after the check, both receipts. Mutants: the bump removed; the reading taken before the check. [proof: mutation]
3. [S3] receipts.txt, schema, AGENTS.md, contract §225. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/writer/ -count=1 -timeout 300s -run 'TestEveryLandedWriteBumpsTheCounter|TestAWriteThatLandsDuringTheCheckIsCounted' -v 2>&1 | tee "$out" \
  && grep -qE '^--- (PASS|SKIP): TestEveryLandedWriteBumpsTheCounter \(' "$out" \
  && grep -qE '^--- (PASS|SKIP): TestAWriteThatLandsDuringTheCheckIsCounted \(' "$out" \
  && go test ./internal/writer/ ./internal/mcp/ ./cmd/mrw/ -count=1 -timeout 900s \
  && grep -q '^write drift_writers$' docs/receipts.txt \
  && grep -q '^mcp_write drift_writers$' docs/receipts.txt \
  && grep -q '^# 225\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines internal/rooted \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestEveryLandedWriteBumpsTheCounter` | `internal/writer/writes127_test.go` | two landed writes move the counter by two; a refused one and a dry run do not | none | S1, S2 |
| `TestAWriteThatLandsDuringTheCheckIsCounted` | `internal/writer/writes127_test.go` | a write landed while the first write's check waits on a gate gives the first `DriftWriters` 1; with none, 0 | none | S1, S2 |
| `TestTheCLIReceiptNamesOtherWritersDuringTheCheck` | `cmd/mrw/drift127_test.go` | the human receipt line names N other writes, and nothing when N is 0 (the in-process review of #342) | none | S3 |
| `TestTheMCPTextNamesOtherWritersDuringTheCheck` | `internal/mcp/drift127_test.go` | the MCP text line, the same | none | S3 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | `Writes`, `bumpWrites` |
| 2 — something selects it | `writer.applyCounted`, through `Prepared.Land`, on every landed write; `Verify` after every check that ran |
| 3 — the caller can discover it | the receipt key and line; AGENTS.md |
| 4 — it is used | a Windows peer met the blind spot on 2026-10-02; no telemetry (ADR-009) |

## Invariants

- The exit code is the check's.
- A write is never refused over the counter.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if the counter has to move under a lock other than the write lock.

## Out of Scope

- Naming the other writer's files (permanent: boundary: they are in that writer's receipt)

## Mutation Log
- 2026-10-06 · 49f1a66* · mutant killed · exit 1 · `internal/writer/writer.go` · S2: the bump removed · acceptance-sha256:aa9f9f3dfc0f4595462e7347c5eafee11cd465e1ae4c58139d59a80f6ed7f241
- 2026-10-06 · 49f1a66* · mutant killed · exit 1 · `internal/writer/flow.go` · S2: the counter not read after the check · acceptance-sha256:aa9f9f3dfc0f4595462e7347c5eafee11cd465e1ae4c58139d59a80f6ed7f241

## Verification Log
- 2026-10-06 · 49f1a66* · exit 1 · `set -o pipefail …` · acceptance-sha256:aa9f9f3dfc0f4595462e7347c5eafee11cd465e1ae4c58139d59a80f6ed7f241 · ms:852 · test-lock-sha256:945943d74484ccb530edd2e59d807c35ccc10d641f10b6f4eea9745ddbf36a76 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvd3JpdGVyL3dyaXRlczEyN190ZXN0LmdvCVRlc3RBV3JpdGVUaGF0TGFuZHNEdXJpbmdUaGVDaGVja0lzQ291bnRlZAkyMDMyZTUzMzg4YTRjNDhhMWZlNDI3YWFkODA1NGI0NTExMDIyYTc1ZmM0ZjRhZDJjNzVjYTI5ZGIzOWQ1Y2NlCmJvZHkJaW50ZXJuYWwvd3JpdGVyL3dyaXRlczEyN190ZXN0LmdvCVRlc3RFdmVyeUxhbmRlZFdyaXRlQnVtcHNUaGVDb3VudGVyCWRjYTdhZjY2NGFjMmZkZjJhOWU2M2Y1M2MwZWFkY2IyZjc3M2I5ZjRlM2M3NWMxYWU3M2E2ODVhZGJhMGNlMzI
  ```
  --- last 9 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/writer [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/writer.test]
  internal/writer/writes127_test.go:33:11: undefined: Writes
  internal/writer/writes127_test.go:36:12: undefined: Writes
  internal/writer/writes127_test.go:43:12: undefined: Writes
  internal/writer/writes127_test.go:97:60: v.DriftWriters undefined (type Verified has no field or method DriftWriters)
  internal/writer/writes127_test.go:100:27: v.DriftWriters undefined (type Verified has no field or method DriftWriters)
  internal/writer/writes127_test.go:101:57: v.DriftWriters undefined (type Verified has no field or method DriftWriters)
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/writer [build failed]
  FAIL
  ```
- 2026-10-06 · 49f1a66* · exit 0 · `set -o pipefail …` · acceptance-sha256:aa9f9f3dfc0f4595462e7347c5eafee11cd465e1ae4c58139d59a80f6ed7f241 · ms:42341
- 2026-10-06 · 49f1a66* · exit 0 · `set -o pipefail …` · acceptance-sha256:aa9f9f3dfc0f4595462e7347c5eafee11cd465e1ae4c58139d59a80f6ed7f241 · ms:40226
- 2026-10-06 · 49f1a66* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:aa9f9f3dfc0f4595462e7347c5eafee11cd465e1ae4c58139d59a80f6ed7f241 · ms:0 · test-lock-sha256:7530d5eb679e1f880c4ff8417b17849fa33c50b8e12de30fa548ec3ec10150d6 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvd3JpdGVyL3dyaXRlczEyN190ZXN0LmdvCVRlc3RBV3JpdGVUaGF0TGFuZHNEdXJpbmdUaGVDaGVja0lzQ291bnRlZAkyMDMyZTUzMzg4YTRjNDhhMWZlNDI3YWFkODA1NGI0NTExMDIyYTc1ZmM0ZjRhZDJjNzVjYTI5ZGIzOWQ1Y2NlCmJvZHkJaW50ZXJuYWwvd3JpdGVyL3dyaXRlczEyN190ZXN0LmdvCVRlc3RFdmVyeUxhbmRlZFdyaXRlQnVtcHNUaGVDb3VudGVyCWI1NjQxN2NlYmU5NWZiMjc4N2YxYzlhYTM0ZjE3ZDVmZTc2YWU3OGYxYTRiMzAyZWFmMmU3MTYwMjg2YWU5NmM · test-lock-kind:replace
- 2026-10-06 · aeab3c5* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:aa9f9f3dfc0f4595462e7347c5eafee11cd465e1ae4c58139d59a80f6ed7f241 · ms:0 · test-lock-sha256:36141bd5d65a63fcbc33a93cc50645ac662705d934f3ca2a07aca504cfc357b0 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJY21kL21ydy9kcmlmdDEyN190ZXN0LmdvCVRlc3RUaGVDTElSZWNlaXB0TmFtZXNPdGhlcldyaXRlcnNEdXJpbmdUaGVDaGVjawllYWQ2MTlmMGNlYTM3MzIzMjRhOWQ4ZTRhYTk2ZTFmNTgxZTBjODhiYThjNjM4ZGEwZTNiNzI4ZmYyODliNTE5CmJvZHkJaW50ZXJuYWwvbWNwL2RyaWZ0MTI3X3Rlc3QuZ28JVGVzdFRoZU1DUFRleHROYW1lc090aGVyV3JpdGVyc0R1cmluZ1RoZUNoZWNrCWVmODZhZTUzYTNlYzgwNzI2YzMwZTZlZTBhNDVlMmYzMTU0NjZhNTE3YmYwMzgzMDJmNGJjZjBhNjUzYTU0OTYKYm9keQlpbnRlcm5hbC93cml0ZXIvd3JpdGVzMTI3X3Rlc3QuZ28JVGVzdEFXcml0ZVRoYXRMYW5kc0R1cmluZ1RoZUNoZWNrSXNDb3VudGVkCTIwMzJlNTMzODhhNGM0OGExZmU0MjdhYWQ4MDU0YjQ1MTEwMjJhNzVmYzRmNGFkMmM3NWNhMjlkYjM5ZDVjY2UKYm9keQlpbnRlcm5hbC93cml0ZXIvd3JpdGVzMTI3X3Rlc3QuZ28JVGVzdEV2ZXJ5TGFuZGVkV3JpdGVCdW1wc1RoZUNvdW50ZXIJYjU2NDE3Y2ViZTk1ZmIyNzg3ZjFjOWFhMzRmMTdkNWZlNzZhZTc4ZjFhNGIzMDJlYWYyZTcxNjAyODZhZTk2Yw · test-lock-kind:replace
