# Task ADR-085-T1: hold and promote take the pending lock

**Depends-on:** none
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** unassigned
**Produces:** the `pending.json.lock` hold around `hold`, `promote` and `nameTheAck`'s read
**Consumes:** `state.Hold`
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `concurrent holds lose no span`, `concurrent acks each license their write`, `a contract row drives the binary`, `the engine packages are unchanged`, `go.mod declares one requirement`

## Goal

Two servers on one checkout could each rewrite `pending.json` from a stale copy, so a served page's acknowledgement matched nothing.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/pending085_test.go` | new | 64 concurrent `hold` calls leave 64 spans |
| `internal/mcp/ack.go` | edit | `hold` and `promote` under `state.Hold`; `readPending` for `nameTheAck` |
| `internal/mcp/tools.go` | edit | `nameTheAck` reads through `readPending` |
| `scripts/contract.sh` | edit | §167: eight `mrw mcp` reads at once, each ack then licenses its write |
| `docs/adr/BACKLOG.md` | edit | the entry closed |

## Ordered Steps

1. [S1] Write `TestConcurrentHoldsLoseNoPendingSpan`; it fails on v1.28.0, where concurrent saves drop spans. [proof: mutation]
2. [S2] Take the pending lock in `hold` and `promote`; read through it in `nameTheAck`. [proof: mutation]
3. [S3] Contract §167 drives eight servers at once. [proof: acceptance]

## Acceptance

```bash
set -o pipefail
go test ./internal/mcp/ -count=1 -timeout 180s -run 'TestConcurrentHoldsLoseNoPendingSpan' -v 2>&1 | tee /tmp/adr085-T1.out \
  && missing=$(for t in TestConcurrentHoldsLoseNoPendingSpan; do grep -qE "^--- PASS: $t \(" /tmp/adr085-T1.out || echo "$t"; done) \
  && [ -z "$missing" ] \
  && grep -q '^# 167\. ' scripts/contract.sh \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines internal/iter internal/rooted internal/subproc \
  && [ -z "$(git status --porcelain --untracked-files=all -- internal/read internal/apply internal/plan internal/seen internal/check internal/state internal/lines internal/iter internal/rooted internal/subproc)" ] \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestConcurrentHoldsLoseNoPendingSpan` | `internal/mcp/pending085_test.go` | every concurrently held span is in the store afterwards | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the lock in `hold`, `promote` and `readPending` |
| 2 — something selects it | every `mrw_read` that serves numbered lines and every `ack` |
| 3 — the caller can discover it | an acknowledgement licenses its write however many servers share the checkout |
| 4 — it is used | the review of #240 found the gap by reading; ADR-009 refuses telemetry, so use is not observed |

## Verification Log
(empty until execute)
- 2026-09-27 · 95c74a2* · exit 1 · `set -o pipefail …` · acceptance-sha256:8483e4b321cdfbf649342783dba080d97ccd58a779acd600d5683f2d814e7752 · ms:1182 · test-lock-sha256:643d63241061f7cacf6fd7ba973596869c39de9ee7188573ea438f96b61730d7 · test-lock-b64:Y2hlY2sJMWJiNDk3ZTNlMTNhMTEwNWNmMjRlMzM1OWZhM2VmNzVkZTA4YjY2ZmY4YTI4MzljZDdmOWVhOTc4MjRkOWViMwpib2R5CWludGVybmFsL21jcC9wZW5kaW5nMDg1X3Rlc3QuZ28JVGVzdENvbmN1cnJlbnRIb2xkc0xvc2VOb1BlbmRpbmdTcGFuCTI2YzgxYmY1NGZmYThlYjY3MWI0OTQ1ZmYyYWZkZDMxM2VlNDllOWVkYzlmZmE4OWYzZmY2MWMyZTNmODljZDA
  ```
  --- last 6 line(s) of stdout
  === RUN   TestConcurrentHoldsLoseNoPendingSpan
      pending085_test.go:33: the pending store holds 4 of the 64 spans held concurrently
  --- FAIL: TestConcurrentHoldsLoseNoPendingSpan (0.01s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.078s
  FAIL
  ```
- 2026-09-27 · 95c74a2* · exit 0 · `set -o pipefail …` · acceptance-sha256:8483e4b321cdfbf649342783dba080d97ccd58a779acd600d5683f2d814e7752 · ms:1045
- 2026-09-27 · 95c74a2* · exit 0 · `set -o pipefail …` · acceptance-sha256:8483e4b321cdfbf649342783dba080d97ccd58a779acd600d5683f2d814e7752 · ms:646

## Mutation Log
(empty until execute)
- 2026-09-27 · 95c74a2* · mutant killed · exit 1 · `internal/mcp/ack.go` · hold rewrites the store with no lock · acceptance-sha256:8483e4b321cdfbf649342783dba080d97ccd58a779acd600d5683f2d814e7752 · covers:concurrent holds lose no span

## Invariants

- Nothing takes the pending lock while holding a ledger lock.
- A pending store that does not parse still licenses nothing (ADR-039).

## Risks

- `promote`'s half has no dedicated fixture; it shares the lock and the load-change-save shape with `hold`, and §167's writes go through it.

## Out of Scope

- `seen.IsStale` (permanent: boundary: the record's Context explains why a single-write save leaves nothing to misread)

## Stop Condition

The fence exits 0 and contract §167 passes.
