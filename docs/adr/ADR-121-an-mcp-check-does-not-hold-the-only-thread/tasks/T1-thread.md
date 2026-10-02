# Task ADR-121-T1: the loop released during a check, progress while a call runs

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** M
**Owner:** Zy
**Produces:** `mcp.callHooks`, the released loop, `notifications/progress`
**Consumes:** none
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `the loop released during a check, progress while a call runs`

## Goal

`mrw mcp` answers a ping, a read or a write that arrives while a write's check runs, keeps every other answer in request order, and sends progress for a call that asks for it.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/mcp.go` | edit | `Serve` dispatches each request and waits for it or its release; one writer; progress |
| `internal/mcp/tools.go` | edit | `writeTool` releases `gate` and the loop across `Verify`; the `gate` note |
| `internal/mcp/thread121_test.go` | add | the tests |
| `AGENTS.md`, `README.md`, `docs/adr/BACKLOG.md` | edit | say so; the deferral |
| `scripts/contract.sh` | edit | §220 |

## Ordered Steps

1. [S1] Write `TestAPingIsAnsweredWhileAWritesCheckRuns`, `TestProgressIsSentWhileACallRunsAndNotAfter`, `TestQuickAnswersKeepTheirOrder`, `TestServeAnswersACallInFlightAtEndOfInput`, `TestAModernWriteKeepsItsDecorationWhenACallRunsDuringItsCheck`. Confirm RED. [proof: mutation]
2. [S2] The loop, the writer, the release around `Verify`, progress. Mutants: no release before `Verify`; the ticker not stopped before the answer; `Serve` returning without waiting; the per-call values not restored after `Verify`. [proof: mutation]
3. [S3] Docs and contract §220: a write whose check sleeps, then a ping — the ping's answer arrives first; the pair, `check: false`, answers in order. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/mcp/ -count=1 -timeout 600s -run 'TestAPingIsAnsweredWhileAWritesCheckRuns|TestProgressIsSentWhileACallRunsAndNotAfter|TestQuickAnswersKeepTheirOrder|TestServeAnswersACallInFlightAtEndOfInput|TestAModernWriteKeepsItsDecorationWhenACallRunsDuringItsCheck|TestTheLoopIsReleasedOnlyWhenACheckOrStepRuns' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAPingIsAnsweredWhileAWritesCheckRuns \(' "$out" \
  && grep -qE '^--- PASS: TestProgressIsSentWhileACallRunsAndNotAfter \(' "$out" \
  && grep -qE '^--- PASS: TestQuickAnswersKeepTheirOrder \(' "$out" \
  && grep -qE '^--- PASS: TestServeAnswersACallInFlightAtEndOfInput \(' "$out" \
  && grep -qE '^--- PASS: TestAModernWriteKeepsItsDecorationWhenACallRunsDuringItsCheck \(' "$out" \
  && grep -qE '^--- PASS: TestTheLoopIsReleasedOnlyWhenACheckOrStepRuns \(' "$out" \
  && go test -race ./internal/mcp/ -count=1 -timeout 900s \
  && go test ./cmd/mrw/ -count=1 -timeout 900s \
  && grep -q '^# 220\. ' scripts/contract.sh \
  && [ -z "$(gofmt -l internal cmd)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines internal/rooted \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAPingIsAnsweredWhileAWritesCheckRuns` | `internal/mcp/thread121_test.go` | a ping sent after a write whose check sleeps is answered before the write | none | S1, S2 |
| `TestProgressIsSentWhileACallRunsAndNotAfter` | `internal/mcp/thread121_test.go` | progress with the token while the write runs, none after its answer | none | S1, S2 |
| `TestQuickAnswersKeepTheirOrder` | `internal/mcp/thread121_test.go` | requests with no check are answered in the order they arrived | none | S1, S2 |
| `TestServeAnswersACallInFlightAtEndOfInput` | `internal/mcp/thread121_test.go` | input closed during a check: the write is still answered | none | S1, S2 |
| `TestAModernWriteKeepsItsDecorationWhenACallRunsDuringItsCheck` | `internal/mcp/thread121_test.go` | a read during a modern write's check does not strip the write's modern fields | none | S1, S2 |
| `TestTheLoopIsReleasedOnlyWhenACheckOrStepRuns` | `internal/mcp/thread121_test.go` | check: false, prose and dry-run writes keep their place; a checked write releases | none | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the released loop, the progress ticker |
| 2 — something selects it | `writeTool` releases before `Verify`; `handle` starts the ticker for a progress token |
| 3 — the caller can discover it | the protocol answers; AGENTS.md and README |
| 4 — it is used | BACKLOG "From ADR-113"; the plan's amendment |

## Mutation Log
- 2026-10-03 · f2923cb* · mutant killed · exit 1 · `internal/mcp/tools.go` · S2: a write releases the loop as its check starts · acceptance-sha256:59633e0bdb7730731526aff6ac8a39bf3f74ba4a299965d5002cfa03b2a588f8
- 2026-10-03 · f2923cb* · mutant killed · exit 1 · `internal/mcp/mcp.go` · S2: progress stops before the answer is written · acceptance-sha256:59633e0bdb7730731526aff6ac8a39bf3f74ba4a299965d5002cfa03b2a588f8
- 2026-10-03 · f2923cb* · mutant killed · exit 1 · `internal/mcp/mcp.go` · S2: Serve answers every call in flight before it returns · acceptance-sha256:59633e0bdb7730731526aff6ac8a39bf3f74ba4a299965d5002cfa03b2a588f8
- 2026-10-03 · f2923cb* · mutant inconclusive · exit 1 · `internal/mcp/tools.go` · S2: the per-call values are restored after the check · acceptance-sha256:59633e0bdb7730731526aff6ac8a39bf3f74ba4a299965d5002cfa03b2a588f8
  ```
  the fence failed on a build/parse error, not an assertion
  ```
- 2026-10-03 · f2923cb* · mutant killed · exit 1 · `internal/mcp/tools.go` · S2: the per-call era is restored after the check (corrects the inconclusive row above, which did not compile) · acceptance-sha256:59633e0bdb7730731526aff6ac8a39bf3f74ba4a299965d5002cfa03b2a588f8
- 2026-10-03 · e57f158* · mutant killed · exit 1 · `internal/mcp/tools.go` · S2: only a write whose check or step runs releases the loop (the review of #327) · acceptance-sha256:8c2bb40a39b2bcd96235da29498b3a281cd9529d22e2707c6b86113e40367de1

## Invariants

- Every ledger read or write happens under `gate`; no line is written while another is half written.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if `-race` reports a race the release introduces that `gate` cannot cover.

## Out of Scope

- Cancellation (deferred: docs/adr/BACKLOG.md "From ADR-121")

## Verification Log
- 2026-10-03 · f2923cb* · exit 1 · `set -o pipefail …` · acceptance-sha256:638463197a3e535c874cb50babce391447f5610b133e24e3c440124c6b01cf1b · ms:1130 · test-lock-sha256:5114d8e4d7e7803554873c195bf570e9e361a2c1cd841b0ce374c2ef265119a1 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvbWNwL3RocmVhZDEyMV90ZXN0LmdvCVRlc3RBUGluZ0lzQW5zd2VyZWRXaGlsZUFXcml0ZXNDaGVja1J1bnMJYjIzMzY2MzA4MDUwNDViM2Q0ZjkxMDgzNWEyOGFmMmZiYTlkODQzNGY4YjFkMWQzZmRlOTQzODRiOThmMTFkMgpib2R5CWludGVybmFsL21jcC90aHJlYWQxMjFfdGVzdC5nbwlUZXN0UHJvZ3Jlc3NJc1NlbnRXaGlsZUFDYWxsUnVuc0FuZE5vdEFmdGVyCTc5MDZhYjdkZWMzNDkxMDZhMGVmMjgwOTliMzliZmJmMDVkYmFmNjhmM2Q1NzdiMWVkMzQ4ZWMzMGQ1NWI5OWYKYm9keQlpbnRlcm5hbC9tY3AvdGhyZWFkMTIxX3Rlc3QuZ28JVGVzdFF1aWNrQW5zd2Vyc0tlZXBUaGVpck9yZGVyCTJjZmM3YWQxZmEyYjMxMTQ5YjQ3YTlhNGEyNTIzMmMwODNmNGFkZDgwOWY5YWFiMTc1NWFkNDA3Zjc1ZWRhMDUKYm9keQlpbnRlcm5hbC9tY3AvdGhyZWFkMTIxX3Rlc3QuZ28JVGVzdFNlcnZlQW5zd2Vyc0FDYWxsSW5GbGlnaHRBdEVuZE9mSW5wdXQJOGRhMGUwZGU1ZTg3ODE4MTA1YmRhMjY4NmM3OTEzNzZhOGQ3YmE1M2E4YTE2N2QzODdkMTEyNDU3NTdjMTc4Mw
  ```
  --- last 6 line(s) of stdout
  # github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp [github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp.test]
  internal/mcp/thread121_test.go:121:9: undefined: progressEvery
  internal/mcp/thread121_test.go:122:2: undefined: progressEvery
  internal/mcp/thread121_test.go:123:21: undefined: progressEvery
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp [build failed]
  FAIL
  ```
- 2026-10-03 · f2923cb* · exit 0 · `set -o pipefail …` · acceptance-sha256:59633e0bdb7730731526aff6ac8a39bf3f74ba4a299965d5002cfa03b2a588f8 · ms:92787
- 2026-10-03 · f2923cb* · exit 0 · `set -o pipefail …` · acceptance-sha256:59633e0bdb7730731526aff6ac8a39bf3f74ba4a299965d5002cfa03b2a588f8 · ms:94129
- 2026-10-03 · f2923cb* · exit 0 · `set -o pipefail …` · acceptance-sha256:59633e0bdb7730731526aff6ac8a39bf3f74ba4a299965d5002cfa03b2a588f8 · ms:86401
- 2026-10-03 · f2923cb* · exit 0 · `set -o pipefail …` · acceptance-sha256:59633e0bdb7730731526aff6ac8a39bf3f74ba4a299965d5002cfa03b2a588f8 · ms:90412
- 2026-10-03 · f2923cb* · exit 0 · `set -o pipefail …` · acceptance-sha256:59633e0bdb7730731526aff6ac8a39bf3f74ba4a299965d5002cfa03b2a588f8 · ms:86206
- 2026-10-03 · e57f158* · exit 0 · `adr-verify --relock --replace-hashes` · acceptance-sha256:8c2bb40a39b2bcd96235da29498b3a281cd9529d22e2707c6b86113e40367de1 · ms:0 · test-lock-sha256:164acc1864494fbec993a818fa8782ab4da30005d6f6a174a58abd8e6ee6827c · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvbWNwL3RocmVhZDEyMV90ZXN0LmdvCVRlc3RBTW9kZXJuV3JpdGVLZWVwc0l0c0RlY29yYXRpb25XaGVuQUNhbGxSdW5zRHVyaW5nSXRzQ2hlY2sJNDhhZWIyYmU3NjM5YmVkMDM2ZmE1ZmRiN2I2OWQ3NWQyNzA0YTRmM2U3ZDE3NzM4MGEwNjI1ZGQ1MjQ4NWNiNQpib2R5CWludGVybmFsL21jcC90aHJlYWQxMjFfdGVzdC5nbwlUZXN0QVBpbmdJc0Fuc3dlcmVkV2hpbGVBV3JpdGVzQ2hlY2tSdW5zCWIyMzM2NjMwODA1MDQ1YjNkNGY5MTA4MzVhMjhhZjJmYmE5ZDg0MzRmOGIxZDFkM2ZkZTk0Mzg0Yjk4ZjExZDIKYm9keQlpbnRlcm5hbC9tY3AvdGhyZWFkMTIxX3Rlc3QuZ28JVGVzdFByb2dyZXNzSXNTZW50V2hpbGVBQ2FsbFJ1bnNBbmROb3RBZnRlcgk3OTA2YWI3ZGVjMzQ5MTA2YTBlZjI4MDk5YjM5YmZiZjA1ZGJhZjY4ZjNkNTc3YjFlZDM0OGVjMzBkNTViOTlmCmJvZHkJaW50ZXJuYWwvbWNwL3RocmVhZDEyMV90ZXN0LmdvCVRlc3RRdWlja0Fuc3dlcnNLZWVwVGhlaXJPcmRlcgkyY2ZjN2FkMWZhMmIzMTE0OWI0N2E5YTRhMjUyMzJjMDgzZjRhZGQ4MDlmOWFhYjE3NTVhZDQwN2Y3NWVkYTA1CmJvZHkJaW50ZXJuYWwvbWNwL3RocmVhZDEyMV90ZXN0LmdvCVRlc3RTZXJ2ZUFuc3dlcnNBQ2FsbEluRmxpZ2h0QXRFbmRPZklucHV0CThkYTBlMGRlNWU4NzgxODEwNWJkYTI2ODZjNzkxMzc2YThkN2JhNTNhOGExNjdkMzg3ZDExMjQ1NzU3YzE3ODMKYm9keQlpbnRlcm5hbC9tY3AvdGhyZWFkMTIxX3Rlc3QuZ28JVGVzdFRoZUxvb3BJc1JlbGVhc2VkT25seVdoZW5BQ2hlY2tPclN0ZXBSdW5zCTVlMjllMWMzZjBlZWI0MzBkMTNmZmIxZjcyY2NkZmQ0NDRkNDgzMmM2MDc4NTIxMjQyNjhiNzE0NzI1MzM2MGI · test-lock-kind:replace
- 2026-10-03 · e57f158* · exit 0 · `set -o pipefail …` · acceptance-sha256:8c2bb40a39b2bcd96235da29498b3a281cd9529d22e2707c6b86113e40367de1 · ms:86323
