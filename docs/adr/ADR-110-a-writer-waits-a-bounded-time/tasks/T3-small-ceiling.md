# Task ADR-110-T3: a refusal before any hunk survives the smallest ceiling

**Depends-on:** T2
**Covers:** none — no spec
**Estimated scope:** S
**Owner:** Zy
**Produces:** `boundedReceipt`'s terminal sentence carries a no-hunk refusal's words
**Consumes:** the write-lock refusal (T2)
**Data dependency:** hermetic
**Proof map:** v1
**Rests-on:** `a refusal before any hunk survives the smallest ceiling`

## Goal

`boundedReceipt` (`internal/mcp/tools.go`) shrinks a receipt that overflows the ceiling and, when even the shrunk one does not fit, answers with a sentence of counts: "0 of 0 hunk(s) failed and nothing was written … send fewer hunks". A write refused before any hunk had a verdict — the write lock held past its wait, T2 — overflowed at the smallest ceiling a write is allowed under (measured: 891 characters), and the sentence dropped the lock, the holder and the remedy (the Codex review of #307). When no hunk has a verdict, the sentence is the refusal itself, cut to fit.

## Affected Files

| File | Change | Why |
|------|--------|-----|
| `internal/mcp/tools.go` | edit | `boundedReceipt`'s terminal branch |
| `internal/mcp/lock110_test.go` | add | the test below |

## Ordered Steps

1. [S1] Write the failing test(s) `TestAWriteLockRefusalSurvivesTheSmallestCeiling`; confirm RED. [proof: mutation]
2. [S2] A refusal with no hunk verdicts answers with its own words, capped. Mutant: the no-hunk branch removed. [proof: mutation]

## Acceptance

```bash
set -o pipefail
out=$(mktemp) \
  && go test ./internal/mcp/ -count=1 -timeout 300s -run 'TestAWriteLockRefusalSurvivesTheSmallestCeiling' -v 2>&1 | tee "$out" \
  && grep -qE '^--- PASS: TestAWriteLockRefusalSurvivesTheSmallestCeiling \(' "$out" \
  && [ -z "$(gofmt -l cmd/mrw internal)" ] \
  && git diff --quiet "$(git merge-base HEAD origin/main)" -- internal/read internal/apply internal/plan internal/check internal/lines internal/rooted \
  && [ "$(grep -cE '^require|^[[:space:]]' go.mod)" = "1" ]
```

## Tests

| Test name | File | Verifies | Covers | Steps |
|-----------|------|----------|--------|-------|
| `TestAWriteLockRefusalSurvivesTheSmallestCeiling` | `internal/mcp/lock110_test.go` | at `minWriteCeiling()`, an `mrw_write` refused by a held write lock answers `isError` with text naming the write lock and the holder's pid | — | S1, S2 |

## Reachability

| Rung | How this task shows it |
|------|------------------------|
| 1 — exists | the code named in Affected Files |
| 2 — something selects it | every `mrw_write` whose receipt overflows the ceiling |
| 3 — the caller can discover it | the refusal names the lock, the holder and the variable |
| 4 — it is used | telemetry is refused (ADR-009); the evidence is the Codex review of #307 |

## Mutation Log
- 2026-10-01 · e7f79fe* · mutant killed · exit 1 · `internal/mcp/tools.go` · the no-hunk branch removed: the refusal at the smallest ceiling says only 0 of 0 hunks · acceptance-sha256:2647c9e047cbfb443f701f7fb6e940800cecb028aa35fdba22ada4c119eb97bb

## Invariants

- A receipt with hunk verdicts takes the same terminal sentence as before.

## Risks

- None beyond the record's.

## Stop Condition

Stop and ask if a locked test must change to pass.

## Out of Scope

- The other ADR-110 tasks, each in its own file.

## Verification Log
- 2026-10-01 · e7f79fe* · exit 1 · `set -o pipefail …` · acceptance-sha256:2647c9e047cbfb443f701f7fb6e940800cecb028aa35fdba22ada4c119eb97bb · ms:331 · test-lock-sha256:df4bf814e31eb026829f8764cede2da9dfe6bf0ee2c60acb2682f65a09fcd934 · test-lock-b64:Y2hlY2tAMgkxYmI0OTdlM2UxM2ExMTA1Y2YyNGUzMzU5ZmEzZWY3NWRlMDhiNjZmZjhhMjgzOWNkN2Y5ZWE5NzgyNGQ5ZWIzCmJvZHkJaW50ZXJuYWwvbWNwL2xvY2sxMTBfdGVzdC5nbwlUZXN0QVdyaXRlTG9ja1JlZnVzYWxTdXJ2aXZlc1RoZVNtYWxsZXN0Q2VpbGluZwk3MTI1NGZhMDMyZTdjMWJkMzU2NzM3N2NiYmQ0M2NkZGNhZTgyYmQzZWJkMzJmNDVhNTI5MDlkY2M5NjcyMjM5
  ```
  --- last 7 line(s) of stdout
  === RUN   TestAWriteLockRefusalSurvivesTheSmallestCeiling
      lock110_test.go:32: a write-lock refusal at a 891-character ceiling lost its words (isError true):
          0 of 0 hunk(s) failed and nothing was written. Naming them takes more than the 891-byte ceiling this server advertises, so they are not listed here. Send fewer hunks in one plan, or use the CLI `mrw write`, which streams and has no such limit.
  --- FAIL: TestAWriteLockRefusalSurvivesTheSmallestCeiling (0.00s)
  FAIL
  FAIL	github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/mcp	0.082s
  FAIL
  ```
- 2026-10-01 · e7f79fe* · exit 0 · `set -o pipefail …` · acceptance-sha256:2647c9e047cbfb443f701f7fb6e940800cecb028aa35fdba22ada4c119eb97bb · ms:1032
